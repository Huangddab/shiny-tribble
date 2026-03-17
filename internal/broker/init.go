package broker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

var (
	_client *nats.Conn
	once    sync.Once
)

func InitializedClient(ctx context.Context) {
	once.Do(func() {
		var err error
		client, err := NewClient()
		if err != nil {
			panic(err)
		}
		_client = client
	})
}

func GetClient() *nats.Conn {
	return _client
}

// Request 请求消息
func Request(subj string, msg []byte, timeout time.Duration) (*nats.Msg, error) {
	if _client == nil {
		return nil, errors.New("broker client not initialized")
	}
	return _client.Request(subj, msg, timeout)
}

// Publish 发布消息
func Publish(subj string, msg []byte) error {
	if _client == nil {
		return errors.New("broker client not initialized")
	}
	return _client.Publish(subj, msg)
}

// Subscribe 订阅消息
func Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	if _client == nil {
		return nil, errors.New("broker client not initialized")
	}
	return _client.Subscribe(subj, cb)
}

// Close 停止客户端
func Close() {
	if _client != nil {
		_client.Close()
		_client = nil // 设为nil，实现fail-fast
	}
}
