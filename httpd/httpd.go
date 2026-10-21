package httpd

import (
	"context"
	"data-server/httpd/router"
	"data-server/internal/dashboard"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type HttpdService struct {
	ctx       context.Context
	router    *gin.Engine
	server    *http.Server
	dashboard *dashboard.Store
	tileCmd   *exec.Cmd
}

func ensureTileServer(ctx context.Context) *exec.Cmd {
	target := strings.TrimRight(os.Getenv("TILE_SERVER_URL"), "/")
	if target == "" {
		target = strings.TrimRight(viper.GetString("tileserver.url"), "/")
	}
	if target == "" {
		target = "http://127.0.0.1:8081"
	}

	if strings.Contains(target, "127.0.0.1:8081") || strings.Contains(target, "localhost:8081") {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:8081", 300*time.Millisecond)
		if err == nil {
			conn.Close()
			logrus.Info("TileServer is already running on 127.0.0.1:8081")
			return nil
		}

		execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			execDir = "."
		}
		martinPath := filepath.Join(execDir, "services", "tileserver", "martin.exe")
		dataPath := filepath.Join(execDir, "data", "tileserver")

		if _, err := os.Stat(martinPath); err != nil {
			martinPath = filepath.Join("services", "tileserver", "martin.exe")
			dataPath = filepath.Join("data", "tileserver")
		}

		if _, err := os.Stat(martinPath); err == nil {
			logrus.Infof("Starting tile server: %s with data: %s", martinPath, dataPath)
			cmd := exec.CommandContext(ctx, martinPath, "--route-prefix", "/data", "-l", "127.0.0.1:8081", dataPath)
			cmd.Dir = execDir
			if err := cmd.Start(); err != nil {
				logrus.Errorf("failed to start tile server: %v", err)
			} else {
				for i := 0; i < 30; i++ {
					c, e := net.DialTimeout("tcp", "127.0.0.1:8081", 100*time.Millisecond)
					if e == nil {
						c.Close()
						logrus.Info("Tile server is ready on 127.0.0.1:8081")
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
				return cmd
			}
		} else {
			logrus.Warnf("tile server binary not found at %s", martinPath)
		}
	}
	return nil
}

func NewService(parent context.Context) *HttpdService {
	// 设置gin运行模式
	gin.SetMode(viper.GetString("httpd.mode"))
	// 初始化gin引擎
	router := gin.Default()
	// 设置模板加载目录
	router.LoadHTMLGlob("template/**/*")
	// 设置静态资源目录
	router.Static("/assets", "./assets")
	// 是否启用pprof
	if viper.GetBool("httpd.enable_pprof") {
		pprof.Register(router)
	}

	// 确保地图瓦片服务已就绪
	tileCmd := ensureTileServer(parent)

	// 创建HTTP服务器
	httpServer := &http.Server{
		Addr:    viper.GetString("httpd.uri"),
		Handler: router,
	}
	service := HttpdService{
		ctx:       parent,
		router:    router,
		server:    httpServer,
		dashboard: dashboard.NewStore(),
		tileCmd:   tileCmd,
	}
	service.initHandle()
	if err := service.dashboard.Subscribe(); err != nil {
		logrus.Warnf("dashboard MQTT subscriptions unavailable: %v", err)
	}
	return &service
}

func (s *HttpdService) Start() error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("failed to start server: %v", err)
		}
	}()
	go s.dashboard.RunMaintenance(s.ctx, time.Second)
	go runCommandRetentionJob(s.ctx, s.dashboard)
	logrus.Infof("httpd service started on %s", s.server.Addr)
	return nil
}

// runCommandRetentionJob 定期清理超过 dashboard.CommandRetentionPeriod 的指令执行记录
func runCommandRetentionJob(ctx context.Context, store *dashboard.Store) {
	store.PruneExpiredCommands(dashboard.CommandRetentionPeriod)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			store.PruneExpiredCommands(dashboard.CommandRetentionPeriod)
		}
	}
}

func (s *HttpdService) Stop() error {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second*5)
	defer cancel()

	if s.tileCmd != nil && s.tileCmd.Process != nil {
		_ = s.tileCmd.Process.Kill()
	}

	if err := s.server.Shutdown(ctx); err != nil {
		logrus.Errorf("failed to shutdown server: %v", err)
		return err
	}
	return nil
}

// initHandle 初始化HTTP路由和处理程序
func tileProxyHandler() gin.HandlerFunc {
	target := strings.TrimRight(os.Getenv("TILE_SERVER_URL"), "/")
	if target == "" {
		target = strings.TrimRight(viper.GetString("tileserver.url"), "/")
	}
	if target == "" {
		target = "http://127.0.0.1:8081"
	}
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		logrus.Fatalf("invalid TILE_SERVER_URL: %q", target)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logrus.Errorf("tile proxy error: %v", err)
		http.Error(w, "map tile service unavailable", http.StatusBadGateway)
	}
	return func(c *gin.Context) {
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/tiles")
		c.Request.URL.Path = strings.TrimSuffix(c.Request.URL.Path, ".png")
		if c.Request.URL.Path == "" {
			c.Request.URL.Path = "/"
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (s *HttpdService) initHandle() {
	s.router.Any("/tiles/*path", tileProxyHandler())
	s.router.Any("/data/*path", tileProxyHandler())
	s.router.GET("/", router.RootRedirect())
	s.router.GET("/api/setup/status", router.SetupStatus())
	s.router.POST("/api/setup/initialize", router.SetupInitialize())
	s.router.POST("/api/auth/login", router.Login(s.dashboard))
	s.router.POST("/api/auth/logout", router.RequireAuth(), router.Logout(s.dashboard))
	s.router.GET("/api/auth/me", router.RequireAuth(), router.Me())
	map_router := s.router.Group("/map")
	{
		map_router.GET("", router.DashboardView())
	}
	dashboardRouter := s.router.Group("/api/dashboard", router.RequireAuth())
	dashboardRouter.GET("/snapshot", router.DashboardSnapshot(s.dashboard))
	dashboardRouter.GET("/devices", router.DashboardDevices(s.dashboard))
	dashboardRouter.PATCH("/devices/:device_id", router.RenameDevice(s.dashboard))
	dashboardRouter.DELETE("/devices/:device_id", router.DeleteDevice(s.dashboard))
	dashboardRouter.GET("/groups", router.DashboardGroups(s.dashboard))
	dashboardRouter.POST("/commands", router.DashboardCommand(s.dashboard))
	dashboardRouter.POST("/groups/:group/commands", router.DashboardGroupCommand(s.dashboard))
	dashboardRouter.POST("/groups/:group/trainings", router.StartTraining(s.dashboard))
	dashboardRouter.POST("/trainings/:training_id/end", router.EndTraining(s.dashboard))
	dashboardRouter.POST("/trainings/:training_id/source", router.UpdateTrainingSource(s.dashboard))
	dashboardRouter.GET("/trainings/history", router.TrainingHistory(s.dashboard))
	dashboardRouter.GET("/commands/history", router.CommandHistory(s.dashboard))
	dashboardRouter.GET("/telemetry/history", router.TelemetryHistory(s.dashboard))
	dashboardRouter.GET("/devices/:device_id/track", router.DeviceTrack(s.dashboard))
	dashboardRouter.GET("/audit/history", router.AuditHistory(s.dashboard))
	dashboardRouter.PUT("/devices/:device_id/group", router.AssignDeviceGroup(s.dashboard))
	dashboardRouter.POST("/groups/:group/positions/share", router.ShareGroupPositions(s.dashboard))
	dashboardRouter.POST("/alerts/:alert_id/confirm", router.ConfirmAlert(s.dashboard))
	dashboardRouter.GET("/alerts/history", router.AlertHistory(s.dashboard))
	dashboardRouter.GET("/alerts/export", router.ExportAlertHistory(s.dashboard))
	dashboardRouter.GET("/users", router.RequireAdmin(), router.UserList())
	dashboardRouter.POST("/users", router.RequireAdmin(), router.UserCreate())
	dashboardRouter.DELETE("/users/:username", router.RequireAdmin(), router.UserDelete())
}
