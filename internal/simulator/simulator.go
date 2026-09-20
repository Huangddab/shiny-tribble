package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"data-server/internal/mqtt"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Broker        string
	ClientID      string
	Username      string
	Password      string
	Devices       []string
	Interval      time.Duration
	AlarmEvery    time.Duration
	AlarmDuration time.Duration
	Pollutant     string
	PollutantConc float64
}

type device struct {
	id       string
	mode     string
	lat      float64
	lng      float64
	battery  int
	sequence int64
	alarm    *alarm
}

type alarm struct {
	startedAt int64
	maxConc   float64
	lastConc  float64
}

type notifyEnvelope struct {
	ID        string          `json:"id"`
	Type      int             `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
}

type eventEnvelope struct {
	Type      int          `json:"type"`
	Timestamp int64        `json:"timestamp"`
	Message   eventMessage `json:"message"`
}

type eventMessage struct {
	ID       string   `json:"id,omitempty"`
	Result   string   `json:"result,omitempty"`
	Reason   string   `json:"reason,omitempty"`
	Names    []string `json:"names,omitempty"`
	Conc     float64  `json:"conc,omitempty"`
	Duration int      `json:"duration,omitempty"`
	MaxConc  float64  `json:"max_conc,omitempty"`
}

func Run(ctx context.Context, config Config) error {
	if len(config.Devices) == 0 {
		return fmt.Errorf("at least one simulator device is required")
	}
	if config.Interval <= 0 {
		config.Interval = 2 * time.Second
	}
	if config.AlarmEvery > 0 && config.AlarmDuration <= 0 {
		config.AlarmDuration = 20 * time.Second
	}
	if config.Pollutant == "" {
		config.Pollutant = "DMMP"
	}
	if config.PollutantConc <= 0 {
		config.PollutantConc = 3.8
	}
	client, err := mqtt.Connect(mqtt.Config{Broker: config.Broker, ClientID: config.ClientID, Username: config.Username, Password: config.Password})
	if err != nil {
		return err
	}
	defer client.Close()

	simulator := &Simulator{client: client, devices: make(map[string]*device), seenCommands: make(map[string]map[string]eventMessage), config: config}
	for index, id := range config.Devices {
		simulator.devices[id] = &device{id: id, mode: "monitor", lat: 22.5431 + float64(index)*0.0005, lng: 114.0579 + float64(index)*0.0005, battery: 100}
	}
	if err := client.Subscribe("/chem/+/notify", 1, simulator.handleNotify); err != nil {
		return fmt.Errorf("subscribe device notify: %w", err)
	}
	if err := client.Subscribe("/chem/notify", 1, simulator.handleNotify); err != nil {
		return fmt.Errorf("subscribe broadcast notify: %w", err)
	}

	logrus.Infof("mqtt simulator started: devices=%d interval=%s", len(simulator.devices), config.Interval)
	return simulator.loop(ctx)
}

type Simulator struct {
	client       *mqtt.Client
	devices      map[string]*device
	seenCommands map[string]map[string]eventMessage
	config       Config
	mu           sync.Mutex
}

func (simulator *Simulator) loop(ctx context.Context) error {
	ticker := time.NewTicker(simulator.config.Interval)
	defer ticker.Stop()
	var alarmTicker *time.Ticker
	var alarmChannel <-chan time.Time
	if simulator.config.AlarmEvery > 0 {
		alarmTicker = time.NewTicker(simulator.config.AlarmEvery)
		defer alarmTicker.Stop()
		alarmChannel = alarmTicker.C
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			simulator.publishTelemetry()
			simulator.updateAlarms()
		case <-alarmChannel:
			simulator.startAlarms()
		}
	}
}

func (simulator *Simulator) publishTelemetry() {
	simulator.mu.Lock()
	defer simulator.mu.Unlock()
	for _, device := range simulator.devices {
		device.sequence++
		device.battery--
		if device.battery < 20 {
			device.battery = 100
		}
		payload := map[string]any{
			"timestamp": time.Now().Unix(),
			"message": map[string]any{
				"device_id": device.id,
				"timestamp": time.Now().Unix(),
				"mode":      device.mode,
				"rssi":      -50 - int(device.sequence%15),
				"gnss":      map[string]any{"fixed": true, "lat": device.lat, "lng": device.lng, "speed": 0},
				"sensor": map[string]any{
					"pid":     map[string]any{"conc": device.alarmConc(), "threshold": 5.0, "alarm": device.alarm != nil},
					"battery": map[string]any{"pct": device.battery, "voltage": 8.4},
					"alarm":   map[string]any{"names": device.alarmNames()},
				},
				"gsensor": map[string]any{"fall_detected": false, "magnitude": 1.0},
			},
		}
		data, err := json.Marshal(payload)
		if err != nil {
			continue
		}
		if err := simulator.client.Publish("/chem/telemetry/"+device.id, 0, false, data); err != nil {
			logrus.Warnf("publish telemetry for %s failed: %v", device.id, err)
		}
	}
}

func (simulator *Simulator) handleNotify(_ paho.Client, message paho.Message) {
	var notify notifyEnvelope
	if err := json.Unmarshal(message.Payload(), &notify); err != nil {
		logrus.Warnf("simulator received invalid notify: %v", err)
		return
	}
	deviceID := topicDeviceID(message.Topic())
	if deviceID == "" {
		for id := range simulator.devices {
			simulator.applyNotify(id, notify)
		}
		return
	}
	if deviceID == "notify" {
		return
	}
	if _, ok := simulator.devices[deviceID]; ok {
		simulator.applyNotify(deviceID, notify)
	}
}

func (simulator *Simulator) applyNotify(deviceID string, notify notifyEnvelope) {
	if notify.Type == 3 {
		return
	}
	if notify.ID == "" {
		logrus.Warnf("simulator received ACK-required notify without id")
		return
	}
	simulator.mu.Lock()
	device := simulator.devices[deviceID]
	if device == nil {
		simulator.mu.Unlock()
		return
	}
	if previous := simulator.seenCommands[deviceID][notify.ID]; previous.ID != "" {
		simulator.mu.Unlock()
		simulator.publishACK(deviceID, notify.ID, previous)
		return
	}
	if simulator.seenCommands[deviceID] == nil {
		simulator.seenCommands[deviceID] = make(map[string]eventMessage)
	}
	result := eventMessage{ID: notify.ID, Result: "success"}
	if notify.Type == 4 {
		var message struct {
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(notify.Message, &message); err != nil || (message.Mode != "monitor" && message.Mode != "training") {
			result.Result = "failed"
			result.Reason = "invalid mode"
		} else {
			device.mode = message.Mode
		}
	}
	simulator.seenCommands[deviceID][notify.ID] = result
	simulator.mu.Unlock()
	simulator.publishACK(deviceID, notify.ID, result)
}

func (simulator *Simulator) publishACK(deviceID, id string, result eventMessage) {
	data, err := json.Marshal(eventEnvelope{Type: 2, Timestamp: time.Now().Unix(), Message: result})
	if err != nil {
		return
	}
	if err := simulator.client.Publish("/chem/events/"+deviceID, 1, false, data); err != nil {
		logrus.Warnf("publish ACK for %s failed: %v", deviceID, err)
	}
}

func (simulator *Simulator) startAlarms() {
	simulator.mu.Lock()
	defer simulator.mu.Unlock()
	for _, device := range simulator.devices {
		if device.alarm != nil {
			continue
		}
		device.alarm = &alarm{startedAt: time.Now().Unix()}
		simulator.publishAlarmLocked(device)
	}
}

func (simulator *Simulator) updateAlarms() {
	simulator.mu.Lock()
	defer simulator.mu.Unlock()
	for _, device := range simulator.devices {
		if device.alarm == nil {
			continue
		}
		if time.Since(time.Unix(device.alarm.startedAt, 0)) >= simulator.config.AlarmDuration {
			alarmState := device.alarm
			device.alarm = nil
			data, _ := json.Marshal(eventEnvelope{Type: 1, Timestamp: time.Now().Unix(), Message: eventMessage{Names: []string{simulator.config.Pollutant}, Duration: int(time.Now().Unix() - alarmState.startedAt), MaxConc: alarmState.maxConc}})
			_ = simulator.client.Publish("/chem/events/"+device.id, 1, false, data)
			continue
		}
		simulator.publishAlarmLocked(device)
	}
}

func (simulator *Simulator) publishAlarmLocked(device *device) {
	conc := simulator.config.PollutantConc + float64(device.sequence%6)*0.7
	if device.alarm != nil {
		device.alarm.lastConc = conc
		device.alarm.maxConc = math.Max(device.alarm.maxConc, conc)
	}
	data, _ := json.Marshal(eventEnvelope{Type: 0, Timestamp: time.Now().Unix(), Message: eventMessage{Names: []string{simulator.config.Pollutant}, Conc: conc}})
	_ = simulator.client.Publish("/chem/events/"+device.id, 1, false, data)
}

func (device *device) alarmConc() float64 {
	if device.alarm == nil {
		return 0
	}
	return device.alarm.lastConc
}

func (device *device) alarmNames() []string {
	if device.alarm == nil {
		return []string{}
	}
	return []string{"DMMP"}
}

func topicDeviceID(topic string) string {
	parts := strings.Split(strings.Trim(topic, "/"), "/")
	if len(parts) == 3 && parts[0] == "chem" && parts[2] == "notify" {
		return parts[1]
	}
	return ""
}

func DeviceIDs(spec string, count int) []string {
	if strings.TrimSpace(spec) != "" {
		parts := strings.Split(spec, ",")
		ids := make([]string, 0, len(parts))
		for _, part := range parts {
			if id := strings.TrimSpace(part); id != "" {
				ids = append(ids, id)
			}
		}
		return ids
	}
	if count <= 0 {
		count = 1
	}
	ids := make([]string, count)
	for index := range ids {
		ids[index] = "sim-" + strconv.Itoa(index+1)
	}
	return ids
}
