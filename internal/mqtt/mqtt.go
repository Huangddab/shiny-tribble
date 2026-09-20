package mqtt

import (
	"context"
	"errors"
	"sync"

	paho "github.com/eclipse/paho.mqtt.golang"
)

var (
	_client *Client
	once    sync.Once
)

// InitializedClient 初始化 MQTT 客户端
func InitializedClient(ctx context.Context) {
	once.Do(func() {
		var err error
		_client, err = NewClient()
		if err != nil {
			panic(err)
		}
	})
}

// GetClient 获取 MQTT 客户端
func GetClient() *Client {
	return _client
}

// Stop 关闭 MQTT 客户端
func Stop() {
	if _client == nil {
		return
	}

	_client.Close()
	_client = nil
}

// Publish 发布消息
func Publish(topic string, qos byte, retained bool, payload []byte) error {
	if _client == nil {
		return errors.New("mqtt client not initialized")
	}

	return _client.Publish(topic, qos, retained, payload)
}

// Subscribe 订阅消息
func Subscribe(topic string, qos byte, callback paho.MessageHandler) error {
	if _client == nil {
		return errors.New("mqtt client not initialized")
	}

	return _client.Subscribe(topic, qos, callback)
}
