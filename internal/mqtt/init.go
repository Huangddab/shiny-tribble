package mqtt

import (
	"fmt"
	"sync"
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
	client        paho.Client
	mu            sync.RWMutex
	subscriptions map[string]subscription
}

type subscription struct {
	qos      byte
	callback paho.MessageHandler
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

	client := &Client{subscriptions: make(map[string]subscription)}
	options.SetOnConnectHandler(func(paho.Client) {
		client.resubscribe()
	})
	mqttClient := paho.NewClient(options)
	client.client = mqttClient

	token := mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connect mqtt broker: %w", token.Error())
	}

	return client, nil
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
	if err := token.Error(); err != nil {
		return err
	}
	client.mu.Lock()
	client.subscriptions[topic] = subscription{qos: qos, callback: callback}
	client.mu.Unlock()
	return nil
}

func (client *Client) resubscribe() {
	client.mu.RLock()
	subscriptions := make(map[string]subscription, len(client.subscriptions))
	for topic, subscription := range client.subscriptions {
		subscriptions[topic] = subscription
	}
	client.mu.RUnlock()
	for topic, subscription := range subscriptions {
		token := client.client.Subscribe(topic, subscription.qos, subscription.callback)
		if token.Wait() && token.Error() != nil {
			continue
		}
	}
}
