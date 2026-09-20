package cmd

import (
	"context"
	"data-server/httpd"
	"data-server/internal/dashboard"
	"data-server/internal/database"
	"data-server/internal/mqtt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	httpdCommand = &cobra.Command{
		Use:   "httpd",
		Short: "HTTP daemon commands",
		Long:  "Commands for managing the HTTP daemon",
		Run:   runHttpdCommand,
	}
)

func initHttpdCommand() {
	httpdCommand.Flags().String("httpd.uri", ":8080", "API service address")
	httpdCommand.Flags().String("httpd.mode", "release", "Run mode (debug|release|test)")
	httpdCommand.Flags().Bool("httpd.enable_pprof", false, "Enable performance profiling")

	if err := viper.BindPFlags(httpdCommand.Flags()); err != nil {
		logrus.Errorf("failed to bind httpd flags: %v", err)
	}

	// Add httpd command to root
	rootCmd.AddCommand(httpdCommand)
}

func runHttpdCommand(cmd *cobra.Command, args []string) {
	// ----- Context -----
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化MQTT服务
	mqtt.InitializedClient(ctx)
	defer mqtt.Stop()

	// 数据库不可用时保留内存模式，不能阻断实时服务
	database.InitializedDatabase(ctx)
	defer database.Stop()
	dashboard.CleanupLegacyData()

	// 初始化HTTP服务
	httpd.InitializeService(ctx)
	defer httpd.Stop()

	// 设置信号处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 等待终止信号
	<-quit
	logrus.Info("httpd command received shutdown signal, exiting...")
}
