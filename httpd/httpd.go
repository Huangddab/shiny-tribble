package httpd

import (
	"context"
	"data-server/httpd/router"
	"data-server/internal/dashboard"
	"net/http"
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
	logrus.Infof("httpd service started on %s", s.server.Addr)
	return nil
}

func (s *HttpdService) Stop() error {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second*5)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		logrus.Errorf("failed to shutdown server: %v", err)
		return err
	}
	return nil
}

// initHandle 初始化HTTP路由和处理程序
func (s *HttpdService) initHandle() {
	s.router.POST("/api/auth/login", router.Login(s.dashboard))
	s.router.POST("/api/auth/logout", router.RequireAuth(), router.Logout(s.dashboard))
	map_router := s.router.Group("/map")
	{
		map_router.GET("", router.DashboardView())
	}
	dashboardRouter := s.router.Group("/api/dashboard", router.RequireAuth())
	dashboardRouter.GET("/snapshot", router.DashboardSnapshot(s.dashboard))
	dashboardRouter.GET("/devices", router.DashboardDevices(s.dashboard))
	dashboardRouter.PATCH("/devices/:device_id", router.RenameDevice(s.dashboard))
	dashboardRouter.GET("/groups", router.DashboardGroups(s.dashboard))
	dashboardRouter.POST("/commands", router.DashboardCommand(s.dashboard))
	dashboardRouter.POST("/groups/:group/commands", router.DashboardGroupCommand(s.dashboard))
	dashboardRouter.POST("/groups/:group/trainings", router.StartTraining(s.dashboard))
	dashboardRouter.POST("/trainings/:training_id/end", router.EndTraining(s.dashboard))
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
}
