package broker

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewClient() (*nats.Conn, error) {
	url := viper.GetString("nats.uri")
	if url == "" {
		return nil, fmt.Errorf("nats connection uri not configured")
	}
	return Connect(url)
}

func Connect(url string) (*nats.Conn, error) {
	options := nats.GetDefaultOptions()
	options.Url = url
	options.MaxReconnect = -1 // Infinite retries
	options.ReconnectWait = 2 * time.Second
	nc, err := options.Connect()
	if err != nil {
		logrus.Errorf("Failed to connect to %s: %v", url, err)
		return nil, err
	}
	logrus.Infof("Connected to NATS server: %s", url)
	return nc, nil
}
