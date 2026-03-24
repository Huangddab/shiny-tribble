package httpd

import (
	"context"

	"github.com/sirupsen/logrus"
)

var (
	_service *HttpdService
)

// 初始化
func InitializeService(ctx context.Context) {
	_service = NewService(ctx)
	if err := _service.Start(); err != nil {
		logrus.Fatalf("failed to start httpd service: %v", err)
	}
}

// 停止
func Stop() {
	if _service == nil {
		return
	}
	_service.Stop()
}
