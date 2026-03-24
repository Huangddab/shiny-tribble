package httpd

import (
	"context"
	"data-server/httpd/router"
	"net/http"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type HttpdService struct {
	ctx    context.Context
	router *gin.Engine
	server *http.Server
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
		ctx:    parent,
		router: router,
		server: httpServer,
	}
	service.initHandle()
	return &service
}

func (s *HttpdService) Start() error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("failed to start server: %v", err)
		}
	}()
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
	map_router := s.router.Group("/map")
	{
		map_router.GET("", router.MapView())
	}
}
