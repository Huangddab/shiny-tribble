package mqtt

import (
	"fmt"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
)

type Config struct {
	Broker   string
	ClientID string
	Username string
	Password string
}

type Client struct {
	client paho.Client
}

func NewClient() (*Client, error) {
	config := Config{
		Broker:   viper.GetString("mqtt.broker"),
		ClientID: viper.GetString("mqtt.client_id"),
		Username: viper.GetString("mqtt.username"),
		Password: viper.GetString("mqtt.password"),
	}

	return Connect(config)
}

func Connect(config Config) (*Client, error) {
	if config.Broker == "" {
		return nil, fmt.Errorf("mqtt broker is required")
	}

	if config.ClientID == "" {
		return nil, fmt.Errorf("mqtt client_id is required")
	}

	options := paho.NewClientOptions()
	options.AddBroker(config.Broker)
	options.SetClientID(config.ClientID)
	options.SetUsername(config.Username)
	options.SetPassword(config.Password)
	options.SetAutoReconnect(true)
	options.SetConnectRetry(true)
	options.SetConnectRetryInterval(2 * time.Second)

	mqttClient := paho.NewClient(options)

	token := mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connect mqtt broker: %w", token.Error())
	}

	return &Client{
		client: mqttClient,
	}, nil
}

func (client *Client) Close() {
	if client == nil || client.client == nil {
		return
	}

	client.client.Disconnect(1000)
}

func (client *Client) Publish(topic string, qos byte, retained bool, payload []byte) error {
	if client == nil || client.client == nil {
		return fmt.Errorf("mqtt client is not connected")
	}

	token := client.client.Publish(topic, qos, retained, payload)
	token.Wait()
	return token.Error()
}

func (client *Client) Subscribe(topic string, qos byte, callback paho.MessageHandler) error {
	if client == nil || client.client == nil {
		return fmt.Errorf("mqtt client is not connected")
	}

	token := client.client.Subscribe(topic, qos, callback)
	token.Wait()
	return token.Error()
}
