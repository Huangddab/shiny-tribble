package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"data-server/internal/database"
	"data-server/internal/model"
	"data-server/internal/mqtt"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Store struct {
	mu                sync.RWMutex
	devices           map[string]*deviceState
	alerts            map[string]*model.DashboardAlert
	alertStarted      map[string]int64
	alertHistory      []model.DashboardAlert
	commands          map[string]*model.DashboardCommand
	commandDeadlines  map[string]time.Time
	commandTargets    map[string]map[string]struct{}
	trainings         map[string]*model.DashboardTraining
	trainingCommands  map[string]string
	trainingStarted   map[string]time.Time
	sourceSent        map[string]sourceDelivery
	samples           []model.DashboardTelemetrySample
	lastSampleAt      map[string]int64
	lastPositionShare map[string]int64
}

const positionShareInterval = 5 * time.Second

// CommandRetentionPeriod 指令执行记录的最长保留时长，超过后由定时任务清理
const CommandRetentionPeriod = 182 * 24 * time.Hour

type positionShare struct {
	target string
	data   []byte
}

type deviceState struct {
	device   model.DashboardDevice
	lastSeen time.Time
}

func CleanupLegacyData() {
	cleanupLegacyDashboardData()
}

type telemetryEnvelope struct {
	Timestamp int64 `json:"timestamp"`
	Message   struct {
		DeviceID string `json:"device_id"`
		Mode     string `json:"mode"`
		RSSI     int    `json:"rssi"`
		GNSS     struct {
			Fixed bool    `json:"fixed"`
			Lat   float64 `json:"lat"`
			Lng   float64 `json:"lng"`
		} `json:"gnss"`
		Sensor struct {
			PID struct {
				Conc      float64 `json:"conc"`
				Threshold float64 `json:"threshold"`
			} `json:"pid"`
			Battery struct {
				Pct int `json:"pct"`
			} `json:"battery"`
			Alarm struct {
				Names []string `json:"names"`
			} `json:"alarm"`
		} `json:"sensor"`
		GSensor struct {
			FallDetected bool `json:"fall_detected"`
		} `json:"gsensor"`
	} `json:"message"`
}

type eventEnvelope struct {
	Type      int   `json:"type"`
	Timestamp int64 `json:"timestamp"`
	Message   struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	} `json:"message"`
}

func NewStore() *Store {
	store := &Store{devices: make(map[string]*deviceState), alerts: make(map[string]*model.DashboardAlert), alertStarted: make(map[string]int64), commands: make(map[string]*model.DashboardCommand), commandDeadlines: make(map[string]time.Time), commandTargets: make(map[string]map[string]struct{}), trainings: make(map[string]*model.DashboardTraining), trainingCommands: make(map[string]string), lastSampleAt: make(map[string]int64), lastPositionShare: make(map[string]int64)}
	if devices, err := loadDeviceRegistry(); err == nil {
		for _, device := range devices {
			store.devices[device.ID] = &deviceState{device: device}
		}
	} else if database.GetDatabase() != nil {
		logrus.Warnf("load dashboard device registry failed: %v", err)
	}
	if commands, err := loadCommandHistory("", "", 1000); err == nil {
		for index := range commands {
			command := commands[index]
			store.commands[command.ID] = &command
			targets := strings.Split(command.Target, ",")
			targetSet := make(map[string]struct{}, len(targets))
			for _, target := range targets {
				if target = strings.TrimSpace(target); target != "" {
					targetSet[target] = struct{}{}
				}
			}
			store.commandTargets[command.ID] = targetSet
		}
	} else if database.GetDatabase() != nil {
		logrus.Warnf("load dashboard command history failed: %v", err)
	}
	if trainings, err := loadTrainingHistory(1000); err == nil {
		for index := range trainings {
			training := trainings[index]
			store.trainings[training.ID] = &training
		}
	} else if database.GetDatabase() != nil {
		logrus.Warnf("load dashboard training history failed: %v", err)
	}
	store.trainingStarted = make(map[string]time.Time)
	store.sourceSent = make(map[string]sourceDelivery)
	return store
}

func alertKey(deviceID string) string {
	return deviceID
}

func (store *Store) UpdateDeviceName(deviceID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("device name is required")
	}
	store.mu.Lock()
	device := store.devices[deviceID]
	if device == nil {
		store.mu.Unlock()
		return fmt.Errorf("device %s not found", deviceID)
	}
	device.device.Name = name
	updated := device.device
	store.mu.Unlock()
	go persistDevice(updated)
	go persistAudit("system", "device.rename", deviceID, "success", map[string]any{"name": name})
	return nil
}

func (store *Store) DeleteDevice(deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}
	store.mu.Lock()
	if _, ok := store.devices[deviceID]; !ok {
		store.mu.Unlock()
		return fmt.Errorf("device %s not found", deviceID)
	}
	delete(store.devices, deviceID)
	delete(store.alerts, alertKey(deviceID))
	delete(store.alertStarted, alertKey(deviceID))
	delete(store.lastSampleAt, deviceID)
	store.mu.Unlock()
	if err := deletePersistedDevice(deviceID); err != nil && database.GetDatabase() != nil {
		return fmt.Errorf("delete device %s from database: %w", deviceID, err)
	}
	go persistAudit("system", "device.delete", deviceID, "success", nil)
	return nil
}

func (store *Store) Groups() []model.DashboardGroup {
	store.mu.RLock()
	groups := make(map[string]*model.DashboardGroup)
	for _, state := range store.devices {
		group := state.device.Group
		if group == "" || group == "未分组" {
			continue
		}
		if groups[group] == nil {
			groups[group] = &model.DashboardGroup{Name: group, Capacity: 10}
		}
		item := groups[group]
		item.Total++
		if state.device.Status == "offline" {
			item.Offline++
		} else {
			item.Online++
		}
		if state.device.Status == "alert" {
			item.Alerts++
		}
	}
	store.mu.RUnlock()
	result := make([]model.DashboardGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].Name < result[right].Name })
	return result
}

func (store *Store) TelemetryHistory(deviceID string, from, to int64, limit int) ([]model.DashboardTelemetrySample, error) {
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("limit must be between 1 and 1000")
	}
	if history, err := loadTelemetryHistory(deviceID, from, to, limit); err == nil {
		return history, nil
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	history := make([]model.DashboardTelemetrySample, 0, limit)
	for index := len(store.samples) - 1; index >= 0 && len(history) < limit; index-- {
		sample := store.samples[index]
		if deviceID != "" && sample.DeviceID != deviceID || from > 0 && sample.Timestamp < from || to > 0 && sample.Timestamp > to {
			continue
		}
		history = append(history, sample)
	}
	return history, nil
}

func (store *Store) AuditHistory(limit int) ([]model.DashboardAudit, error) {
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("limit must be between 1 and 1000")
	}
	return loadAuditHistory(limit)
}

func (store *Store) RecordAudit(actor, action, target, result string, metadata map[string]any) {
	go persistAudit(actor, action, target, result, metadata)
}

func (store *Store) SendNotify(deviceID string, notifyType int, message any) (string, error) {
	if deviceID == "all" {
		store.mu.RLock()
		targets := make([]string, 0, len(store.devices))
		for id := range store.devices {
			targets = append(targets, id)
		}
		store.mu.RUnlock()
		// "all" 使用协议约定的广播 Topic，而非逐设备单播
		return store.sendNotify(targets, notifyType, message, true)
	}
	return store.sendNotify([]string{deviceID}, notifyType, message, false)
}

func (store *Store) SendGroupNotify(group string, notifyType int, message any) (string, error) {
	store.mu.RLock()
	targets := make([]string, 0)
	for deviceID, state := range store.devices {
		if state.device.Group == group {
			targets = append(targets, deviceID)
		}
	}
	store.mu.RUnlock()
	if len(targets) == 0 {
		return "", fmt.Errorf("group %s has no devices", group)
	}
	return store.sendNotify(targets, notifyType, message, false)
}

func (store *Store) StartTraining(group, name string) (model.DashboardTraining, error) {
	return store.StartTrainingWithSource(group, name, nil)
}

func (store *Store) StartTrainingWithSource(group, name string, source *model.DashboardPollutionSource) (model.DashboardTraining, error) {
	if err := validateTrainingSource(source); err != nil {
		return model.DashboardTraining{}, err
	}
	store.mu.RLock()
	devices := make([]string, 0)
	for deviceID, state := range store.devices {
		if state.device.Group == group {
			devices = append(devices, deviceID)
		}
	}
	store.mu.RUnlock()
	if len(devices) == 0 {
		return model.DashboardTraining{}, fmt.Errorf("group %s has no devices", group)
	}
	commandID, err := store.SendGroupNotify(group, 3, map[string]any{"mode": "training"})
	if err != nil {
		return model.DashboardTraining{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return model.DashboardTraining{}, err
	}
	training := model.DashboardTraining{ID: id.String(), Name: name, Group: group, Devices: devices, Mode: "training", Status: "starting", StartedAt: time.Now().Format("15:04:05"), Source: source}
	store.mu.Lock()
	store.trainings[training.ID] = &training
	store.trainingStarted[training.ID] = time.Now()
	if commandID != "" {
		store.trainingCommands[commandID] = training.ID
	}
	store.mu.Unlock()
	go persistTraining(training)
	go persistAudit("system", "training.start", training.ID, "accepted", map[string]any{"group": group, "name": name})
	return training, nil
}

// resolveTrainingCommandLocked closes the training mode-switch loop: once a
// device ACKs (or times out/fails) the type=3 command that started a
// training, the training status moves off "starting" instead of being stuck
// there forever. Caller must hold store.mu.
func (store *Store) resolveTrainingCommandLocked(commandID, result string) {
	trainingID, ok := store.trainingCommands[commandID]
	if !ok {
		return
	}
	delete(store.trainingCommands, commandID)
	training := store.trainings[trainingID]
	if training == nil || training.Status != "starting" {
		return
	}
	if result == "success" {
		training.Status = "active"
	} else {
		training.Status = "ended"
		training.EndedAt = time.Now().Format("15:04:05")
		delete(store.trainingStarted, training.ID)
	}
	persisted := *training
	go persistTraining(persisted)
}

func (store *Store) EndTraining(trainingID string) (model.DashboardTraining, error) {
	store.mu.Lock()
	training := store.trainings[trainingID]
	if training == nil || training.Status == "ended" {
		store.mu.Unlock()
		return model.DashboardTraining{}, fmt.Errorf("training %s not found or already ended", trainingID)
	}
	group := training.Group
	hasSource := training.Source != nil
	store.mu.Unlock()
	if _, err := store.SendGroupNotify(group, 3, map[string]any{"mode": "monitor"}); err != nil {
		return model.DashboardTraining{}, err
	}
	// 告知设备停止模拟污染源，失败不阻断训练结束
	if hasSource {
		if _, err := store.SendGroupNotify(group, 1, map[string]any{"action": "pollution_source_clear"}); err != nil {
			logrus.Warnf("send pollution source clear to group %s failed: %v", group, err)
		}
	}
	store.mu.Lock()
	training.Mode = "monitor"
	training.Status = "ended"
	training.EndedAt = time.Now().Format("15:04:05")
	delete(store.trainingStarted, training.ID)
	for _, deviceID := range training.Devices {
		delete(store.sourceSent, training.ID+":"+deviceID)
	}
	result := *training
	store.mu.Unlock()
	go persistTraining(result)
	go persistAudit("system", "training.end", result.ID, "accepted", map[string]any{"group": result.Group})
	return result, nil
}

func (store *Store) TrainingHistory(limit int) ([]model.DashboardTraining, error) {
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("limit must be between 1 and 1000")
	}
	if history, err := loadTrainingHistory(limit); err == nil {
		return history, nil
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	history := make([]model.DashboardTraining, 0, len(store.trainings))
	for _, training := range store.trainings {
		history = append(history, *training)
	}
	if len(history) > limit {
		history = history[len(history)-limit:]
	}
	return history, nil
}

func (store *Store) sendNotify(targets []string, notifyType int, message any, broadcast bool) (string, error) {
	if len(targets) == 0 {
		return "", fmt.Errorf("no target devices")
	}
	if err := validateNotifyMessage(notifyType, message); err != nil {
		return "", err
	}
	payload := map[string]any{"type": notifyType, "timestamp": time.Now().Unix(), "message": message}
	commandID := ""
	if notifyType == 0 || notifyType == 1 || notifyType == 3 {
		id, err := uuid.NewV7()
		if err != nil {
			return "", fmt.Errorf("generate command id: %w", err)
		}
		commandID = id.String()
		if notifyType == 1 || notifyType == 3 {
			payload["id"] = commandID
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode notify: %w", err)
	}
	qos := byte(1)
	if notifyType == 2 {
		qos = 0
	}
	if commandID != "" {
		requiresAck := notifyType == 1 || notifyType == 3
		result := "success"
		progress := 100
		if requiresAck {
			result = "pending"
			progress = 0
		}
		results := make([]model.DashboardCommandResult, 0, len(targets))
		targetSet := make(map[string]struct{}, len(targets))
		for _, target := range targets {
			deviceResult := result
			results = append(results, model.DashboardCommandResult{DeviceID: target, Result: deviceResult})
			targetSet[target] = struct{}{}
		}
		command := model.DashboardCommand{ID: commandID, Action: notifyAction(notifyType, message), Target: strings.Join(targets, ","), Result: result, Progress: progress, Results: results}
		command.CreatedAtUnix = time.Now().Unix()
		store.mu.Lock()
		store.commands[commandID] = &command
		store.commandTargets[commandID] = targetSet
		if requiresAck {
			store.commandDeadlines[commandID] = time.Now().Add(10 * time.Second)
		}
		store.mu.Unlock()
		go persistCommand(command, time.Now().Unix())
		go persistAudit("system", "command.send", commandID, "accepted", map[string]any{"type": notifyType, "target": command.Target})
	}
	if broadcast {
		if err := mqtt.Publish("chem/notify", qos, false, data); err != nil {
			return "", err
		}
		return commandID, nil
	}
	for _, target := range targets {
		if err := mqtt.Publish("chem/"+target+"/notify", qos, false, data); err != nil {
			return "", err
		}
	}
	return commandID, nil
}

func (store *Store) CommandHistory(result, target string, limit int) ([]model.DashboardCommand, error) {
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("limit must be between 1 and 1000")
	}
	if history, err := loadCommandHistory(result, target, limit); err == nil {
		return history, nil
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	history := make([]model.DashboardCommand, 0, limit)
	for _, command := range store.commands {
		if result != "" && command.Result != result {
			continue
		}
		if target != "" && !strings.Contains(command.Target, target) {
			continue
		}
		history = append(history, *command)
		if len(history) == limit {
			break
		}
	}
	return history, nil
}

// PruneExpiredCommands 删除超过 retention 时长的指令执行记录，供定时任务调用
func (store *Store) PruneExpiredCommands(retention time.Duration) {
	cutoff := time.Now().Add(-retention).Unix()
	store.mu.Lock()
	for id, command := range store.commands {
		if command.CreatedAtUnix > 0 && command.CreatedAtUnix < cutoff {
			delete(store.commands, id)
			delete(store.commandTargets, id)
			delete(store.commandDeadlines, id)
		}
	}
	store.mu.Unlock()
	deleted, err := deleteExpiredCommands(cutoff)
	if err != nil {
		if database.GetDatabase() != nil {
			logrus.Warnf("prune expired dashboard_commands failed: %v", err)
		}
		return
	}
	if deleted > 0 {
		logrus.Infof("deleted %d expired dashboard_commands records", deleted)
	}
}

func validateNotifyMessage(notifyType int, message any) error {
	switch notifyType {
	case 0:
		value, ok := message.(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("type 0 message must be non-empty text")
		}
	case 1:
		values, ok := message.(map[string]any)
		if !ok {
			return fmt.Errorf("type 1 message must be an object")
		}
		action, ok := values["action"].(string)
		_, validAction := map[string]struct{}{"evacuate": {}, "assemble": {}, "silent_on": {}, "silent_off": {}, "goto": {}, "pollution_source": {}, "pollution_source_clear": {}}[action]
		if !ok || !validAction {
			return fmt.Errorf("type 1 action must be evacuate, assemble, silent_on, silent_off, goto, pollution_source, or pollution_source_clear")
		}
		if action == "goto" && (!isJSONNumber(values["lat"]) || !isJSONNumber(values["lng"])) {
			return fmt.Errorf("type 1 goto requires numeric lat and lng")
		}
		if action == "pollution_source" {
			substance, _ := values["substance"].(string)
			if strings.TrimSpace(substance) == "" || !isJSONNumber(values["conc"]) || values["conc"].(float64) <= 0 {
				return fmt.Errorf("type 1 pollution_source requires substance and positive conc")
			}
		}
	case 2:
		values, ok := message.(map[string]any)
		if !ok {
			return fmt.Errorf("type 2 message must be an object")
		}
		devices, ok := values["devices"].([]any)
		if !ok {
			positions, positionsOK := values["devices"].([]map[string]any)
			if !positionsOK {
				return fmt.Errorf("type 2 devices must contain between 1 and 16 positions")
			}
			devices = make([]any, len(positions))
			for index := range positions {
				devices[index] = positions[index]
			}
		}
		if len(devices) == 0 || len(devices) > 16 {
			return fmt.Errorf("type 2 devices must contain between 1 and 16 positions")
		}
		for _, value := range devices {
			position, ok := value.(map[string]any)
			if !ok {
				return fmt.Errorf("type 2 position must be an object")
			}
			if _, ok := position["device_id"].(string); !ok {
				return fmt.Errorf("type 2 position requires device_id")
			}
			if _, ok := position["lat"].(float64); !ok {
				return fmt.Errorf("type 2 position requires lat")
			}
			if _, ok := position["lng"].(float64); !ok {
				return fmt.Errorf("type 2 position requires lng")
			}
		}
	case 3:
		values, ok := message.(map[string]any)
		if !ok {
			return fmt.Errorf("type 3 message must be an object")
		}
		mode, ok := values["mode"].(string)
		if !ok || (mode != "monitor" && mode != "training") {
			return fmt.Errorf("type 3 mode must be monitor or training")
		}
	default:
		return fmt.Errorf("unsupported notify type %d", notifyType)
	}
	return nil
}

func validateTelemetry(envelope telemetryEnvelope) error {
	if envelope.Timestamp <= 0 {
		return fmt.Errorf("telemetry timestamp is required")
	}
	if envelope.Message.DeviceID == "" {
		return fmt.Errorf("telemetry message.device_id is required")
	}
	if envelope.Message.Mode != "monitor" && envelope.Message.Mode != "training" {
		return fmt.Errorf("telemetry mode must be monitor or training")
	}
	return nil
}

func validateEvent(event eventEnvelope) error {
	if event.Timestamp <= 0 {
		return fmt.Errorf("event timestamp is required")
	}
	switch event.Type {
	case 0:
		if len(event.Message.Names) == 0 && !event.Message.FallDetected {
			return fmt.Errorf("type 0 event requires names or fall_detected")
		}
	case 1:
		if event.Message.Duration < 0 {
			return fmt.Errorf("type 1 duration must not be negative")
		}
	case 2:
		if event.Message.ID == "" || (event.Message.Result != "success" && event.Message.Result != "failed") {
			return fmt.Errorf("type 2 requires id and result success or failed")
		}
	default:
		return fmt.Errorf("unsupported event type %d", event.Type)
	}
	return nil
}

func isJSONNumber(value any) bool {
	number, ok := value.(float64)
	return ok && !math.IsNaN(number) && !math.IsInf(number, 0)
}

func (store *Store) ShareGroupPositions(group string) error {
	store.mu.RLock()
	var devices []map[string]any
	var targetIDs []string
	for deviceID, state := range store.devices {
		if state.device.Group != group || time.Since(state.lastSeen) >= 15*time.Second {
			continue
		}
		devices = append(devices, map[string]any{"device_id": deviceID, "lat": state.device.Lat, "lng": state.device.Lng})
		targetIDs = append(targetIDs, deviceID)
	}
	store.mu.RUnlock()
	if len(devices) == 0 {
		return fmt.Errorf("group %s has no devices", group)
	}
	for _, deviceID := range targetIDs {
		positions := make([]map[string]any, 0, len(devices)-1)
		for _, device := range devices {
			if device["device_id"] == deviceID {
				continue
			}
			positions = append(positions, device)
		}
		if len(positions) > 16 {
			positions = positions[:16]
		}
		message := map[string]any{"devices": positions}
		if _, err := store.sendNotify([]string{deviceID}, 2, message, false); err != nil {
			return err
		}
	}
	return nil
}

func notifyAction(notifyType int, message any) string {
	if notifyType == 1 {
		if values, ok := message.(map[string]any); ok {
			if action, ok := values["action"].(string); ok {
				if action == "pollution_source" {
					return "模拟污染源配置"
				}
				if action == "pollution_source_clear" {
					return "清除模拟污染源"
				}
			}
		}
	}
	switch notifyType {
	case 0:
		return "文本通知"
	case 1:
		return "参数配置"
	case 2:
		return "位置共享"
	case 3:
		return "切换模式"
	}
	return "动作指令"
}

func (store *Store) Subscribe() error {
	if err := mqtt.Subscribe("chem/telemetry/+", 0, store.handleTelemetry); err != nil {
		return err
	}
	return mqtt.Subscribe("chem/events/+", 1, store.handleEvent)
}

func (store *Store) AssignGroup(deviceID, group string) error {
	group = strings.TrimSpace(group)
	if group == "" {
		return fmt.Errorf("group is required")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	device := store.devices[deviceID]
	if device == nil {
		return fmt.Errorf("device %s not found", deviceID)
	}
	count := 0
	for id, state := range store.devices {
		if id != deviceID && state.device.Group == group {
			count++
		}
	}
	if count >= 10 {
		return fmt.Errorf("group %s is full", group)
	}
	device.device.Group = group
	if alert := store.alerts[deviceID]; alert != nil {
		alert.Group = group
	}
	updated := device.device
	go persistDevice(updated)
	go persistAudit("system", "device.assign_group", deviceID, "success", map[string]any{"group": group})
	return nil
}

func (store *Store) ConfirmAlert(alertID string) error {
	store.mu.Lock()
	for _, alert := range store.alerts {
		if alert.ID == alertID {
			alert.Ack = "confirmed"
			confirmed := *alert
			store.mu.Unlock()
			go persistAlertConfirmation(confirmed)
			go persistAudit("system", "alert.confirm", alertID, "success", nil)
			return nil
		}
	}
	store.mu.Unlock()
	return fmt.Errorf("active alert %s not found", alertID)
}

type alertHistoryPage struct {
	Items        []model.DashboardAlert
	Total        int64
	NextBefore   int64
	NextBeforeID string
}

func (store *Store) AlertHistory(deviceID, group string, from, to int64, beforeID string, before int64, limit int) alertHistoryPage {
	if limit <= 0 || limit > 1000 {
		return alertHistoryPage{}
	}
	if history, total, err := loadAlertHistory(deviceID, group, from, to, beforeID, before, limit); err == nil {
		return makeAlertHistoryPage(history, total)
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	history := make([]model.DashboardAlert, 0, limit)
	var total int64
	for index := len(store.alertHistory) - 1; index >= 0; index-- {
		alert := store.alertHistory[index]
		if deviceID != "" && alert.DeviceID != deviceID {
			continue
		}
		if group != "" && alert.Group != group {
			continue
		}
		if from > 0 && alert.EndedAtUnix < from {
			continue
		}
		if to > 0 && alert.EndedAtUnix > to {
			continue
		}
		total++
		if before > 0 && (alert.EndedAtUnix > before || (alert.EndedAtUnix == before && beforeID != "" && alert.ID >= beforeID)) {
			continue
		}
		if int64(len(history)) < int64(limit) {
			history = append(history, alert)
		}
	}
	return makeAlertHistoryPage(history, total)
}

func makeAlertHistoryPage(history []model.DashboardAlert, total int64) alertHistoryPage {
	page := alertHistoryPage{Items: history, Total: total}
	if len(history) > 0 {
		last := history[len(history)-1]
		page.NextBefore = last.EndedAtUnix
		page.NextBeforeID = last.ID
	}
	return page
}

func (store *Store) handleTelemetry(_ paho.Client, message paho.Message) {
	var envelope telemetryEnvelope
	if err := json.Unmarshal(message.Payload(), &envelope); err != nil {
		logrus.Warnf("invalid telemetry from %s: %v", message.Topic(), err)
		return
	}
	if err := validateTelemetry(envelope); err != nil {
		logrus.Warnf("invalid telemetry from %s: %v", message.Topic(), err)
		return
	}
	topicID := topicDeviceID(message.Topic())
	if topicID == "" || envelope.Message.DeviceID != topicID {
		logrus.Warnf("telemetry device identity mismatch from %s", message.Topic())
		return
	}
	deviceID := topicID
	store.applyTelemetry(deviceID, envelope)
}

func (store *Store) applyTelemetry(deviceID string, envelope telemetryEnvelope) {
	now := time.Unix(sanitizeDeviceTimestamp(deviceID, envelope.Timestamp), 0)
	store.mu.Lock()
	device := store.devices[deviceID]
	if device == nil {
		device = &deviceState{device: model.DashboardDevice{ID: deviceID, Name: "设备 " + deviceID, Group: "未分组"}, lastSeen: now}
		store.devices[deviceID] = device
	}
	device.device.Status = "normal"
	device.device.Mode = envelope.Message.Mode
	if envelope.Message.GNSS.Fixed {
		device.device.Lat, device.device.Lng = envelope.Message.GNSS.Lat, envelope.Message.GNSS.Lng
		device.device.PositionValid = true
	}
	device.device.Conc, device.device.Threshold = envelope.Message.Sensor.PID.Conc, envelope.Message.Sensor.PID.Threshold
	device.device.Substance = strings.Join(envelope.Message.Sensor.Alarm.Names, ", ")
	device.device.Battery, device.device.RSSI = envelope.Message.Sensor.Battery.Pct, envelope.Message.RSSI
	device.device.Fall, device.lastSeen = envelope.Message.GSensor.FallDetected, now
	if device.device.Status == "normal" && store.hasActiveAlertsLocked(deviceID) {
		device.device.Status = "alert"
	}
	var sample *model.DashboardTelemetrySample
	if now.Unix()-store.lastSampleAt[deviceID] >= 10 {
		value := model.DashboardTelemetrySample{DeviceID: deviceID, Timestamp: now.Unix(), Mode: device.device.Mode, Training: device.device.Mode == "training", Lat: device.device.Lat, Lng: device.device.Lng, Conc: device.device.Conc, Threshold: device.device.Threshold, Battery: device.device.Battery, RSSI: device.device.RSSI}
		store.lastSampleAt[deviceID] = now.Unix()
		store.samples = append(store.samples, value)
		if len(store.samples) > 10000 {
			store.samples = store.samples[len(store.samples)-10000:]
		}
		sample = &value
	}
	store.mu.Unlock()
	if sample != nil {
		go persistTelemetrySample(*sample)
	}
}

func (store *Store) handleEvent(_ paho.Client, message paho.Message) {
	var event eventEnvelope
	if err := json.Unmarshal(message.Payload(), &event); err != nil {
		logrus.Warnf("invalid event from %s: %v", message.Topic(), err)
		return
	}
	if err := validateEvent(event); err != nil {
		logrus.Warnf("invalid event from %s: %v", message.Topic(), err)
		return
	}
	deviceID := topicDeviceID(message.Topic())
	if deviceID == "" {
		return
	}
	when := sanitizeDeviceTimestamp(deviceID, timestamp(event.Timestamp, time.Now().Unix()))
	store.applyEvent(deviceID, event, when)
}

func (store *Store) applyEvent(deviceID string, event eventEnvelope, when int64) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if event.Type == 2 {
		if command := store.commands[event.Message.ID]; command != nil {
			for index := range command.Results {
				if command.Results[index].DeviceID == deviceID {
					command.Results[index].Result = event.Message.Result
					command.Results[index].Reason = event.Message.Reason
				}
			}
			updateCommandResult(command)
			if command.Result != "pending" {
				delete(store.commandDeadlines, event.Message.ID)
				store.resolveTrainingCommandLocked(event.Message.ID, command.Result)
			}
			go persistCommand(cloneDashboardCommand(*command), time.Now().Unix())
		}
		return
	}
	if event.Type == 0 {
		if len(event.Message.Names) == 0 && !event.Message.FallDetected {
			return
		}
		key := alertKey(deviceID)
		alert := store.alerts[key]
		if alert == nil {
			alert = &model.DashboardAlert{ID: fmt.Sprintf("%s-%d", deviceID, when), DeviceID: deviceID, Group: "未分组", StartedAt: time.Unix(when, 0).Format("15:04:05"), Status: "active", Ack: "pending"}
			store.alerts[key] = alert
			store.alertStarted[key] = when
		}
		if len(event.Message.Names) > 0 {
			alert.Current = event.Message.Conc
			if event.Message.Conc > alert.Max {
				alert.Max = event.Message.Conc
			}
			alert.Substance = strings.Join(event.Message.Names, ", ")
		}
		alert.Fall = alert.Fall || event.Message.FallDetected
		if device := store.devices[deviceID]; device != nil {
			device.device.Status = "alert"
			alert.DeviceName, alert.Group = device.device.Name, device.device.Group
		}
	} else if event.Type == 1 {
		store.finishAlertLocked(alertKey(deviceID), "normal", when, event.Message.Duration, event.Message.MaxConc)
	}
}

// RunMaintenance 周期性地在没有请求到来时主动执行离线判定和指令超时判定
func (store *Store) RunMaintenance(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			store.mu.Lock()
			store.expireLocked(now)
			shares := store.collectPositionSharesLocked(now)
			for id, training := range store.trainings {
				if training.Status == "active" && training.Source != nil && store.trainingStarted[id].IsZero() {
					store.trainingStarted[id] = now
				}
			}
			store.mu.Unlock()
			store.publishPositionShares(shares)
			if now.Unix()%5 == 0 {
				store.updateTrainingSources(now)
			}
		}
	}
}

func (store *Store) collectPositionSharesLocked(now time.Time) []positionShare {
	targetsByGroup := make(map[string][]model.DashboardDevice)
	positionsByGroup := make(map[string][]model.DashboardDevice)
	for _, state := range store.devices {
		if state.device.Group == "" || state.device.Group == "未分组" || now.Sub(state.lastSeen) >= 15*time.Second {
			continue
		}
		targetsByGroup[state.device.Group] = append(targetsByGroup[state.device.Group], state.device)
		if state.device.PositionValid {
			positionsByGroup[state.device.Group] = append(positionsByGroup[state.device.Group], state.device)
		}
	}
	shares := make([]positionShare, 0)
	for group, targets := range targetsByGroup {
		positions := positionsByGroup[group]
		if len(targets) < 2 || len(positions) == 0 || now.Unix()-store.lastPositionShare[group] < int64(positionShareInterval/time.Second) {
			continue
		}
		store.lastPositionShare[group] = now.Unix()
		for _, target := range targets {
			shared := make([]map[string]any, 0, len(positions)-1)
			for _, device := range positions {
				if device.ID == target.ID {
					continue
				}
				shared = append(shared, map[string]any{"device_id": device.ID, "lat": device.Lat, "lng": device.Lng})
			}
			if len(shared) == 0 {
				continue
			}
			if len(shared) > 16 {
				shared = shared[:16]
			}
			data, err := json.Marshal(map[string]any{"type": 2, "timestamp": now.Unix(), "message": map[string]any{"devices": shared}})
			if err == nil {
				shares = append(shares, positionShare{target: target.ID, data: data})
			}
		}
	}
	return shares
}

func (store *Store) publishPositionShares(shares []positionShare) {
	for _, share := range shares {
		if err := mqtt.Publish("chem/"+share.target+"/notify", 0, false, share.data); err != nil {
			logrus.Warnf("publish position share for %s failed: %v", share.target, err)
		}
	}
}

// expireLocked 判定指令超时和设备离线；调用方必须已持有 store.mu 写锁
func (store *Store) expireLocked(now time.Time) {
	for commandID, deadline := range store.commandDeadlines {
		if now.After(deadline) {
			if command := store.commands[commandID]; command != nil && command.Result == "pending" {
				for index := range command.Results {
					if command.Results[index].Result == "pending" {
						command.Results[index].Result = "timeout"
						command.Results[index].Reason = "ack timeout"
					}
				}
				updateCommandResult(command)
				command.TimeoutAtUnix = now.Unix()
				go persistCommand(cloneDashboardCommand(*command), now.Unix())
			}
			delete(store.commandDeadlines, commandID)
		}
	}
	for deviceID, state := range store.devices {
		if state.lastSeen.IsZero() {
			state.device.Status = "offline"
			state.device.LastSeen = "未上报"
			continue
		}
		if now.Sub(state.lastSeen) >= 15*time.Second {
			state.device.Status = "offline"
		}
		state.device.LastSeen = formatLastSeen(now.Sub(state.lastSeen))
		if state.device.Status == "offline" {
			store.finishAlertLocked(alertKey(deviceID), "offline", state.lastSeen.Unix(), 0, 0)
		}
	}
}

func (store *Store) Snapshot() model.DashboardSnapshot {
	now := time.Now()
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(now)
	snapshot := model.DashboardSnapshot{UpdatedAt: now}
	for _, state := range store.devices {
		snapshot.Devices = append(snapshot.Devices, state.device)
		if state.device.Status == "offline" {
			snapshot.Summary.OfflineDevices++
		} else {
			snapshot.Summary.OnlineDevices++
		}
		if state.device.Mode == "training" {
			snapshot.Summary.TrainingDevices++
		}
	}
	snapshot.Summary.TotalDevices = len(snapshot.Devices)
	for _, alert := range store.alerts {
		snapshot.Alerts = append(snapshot.Alerts, *alert)
	}
	snapshot.AlertHistory = append(snapshot.AlertHistory, store.alertHistory...)
	snapshot.Summary.AlertDevices = len(snapshot.Alerts)
	for _, command := range store.commands {
		snapshot.Commands = append(snapshot.Commands, *command)
	}
	for _, training := range store.trainings {
		snapshot.Trainings = append(snapshot.Trainings, *training)
	}
	// map 遍历顺序每次都不同，排序后前端列表才不会在每次刷新时无意义地重新排列
	sort.Slice(snapshot.Devices, func(left, right int) bool { return snapshot.Devices[left].ID < snapshot.Devices[right].ID })
	sort.Slice(snapshot.Commands, func(left, right int) bool {
		if snapshot.Commands[left].CreatedAtUnix != snapshot.Commands[right].CreatedAtUnix {
			return snapshot.Commands[left].CreatedAtUnix > snapshot.Commands[right].CreatedAtUnix
		}
		return snapshot.Commands[left].ID > snapshot.Commands[right].ID
	})
	sort.Slice(snapshot.Trainings, func(left, right int) bool { return snapshot.Trainings[left].ID < snapshot.Trainings[right].ID })
	return snapshot
}

func updateCommandResult(command *model.DashboardCommand) {
	if len(command.Results) == 0 {
		command.Result = "pending"
		return
	}
	completed := 0
	for _, result := range command.Results {
		switch result.Result {
		case "failed":
			command.Result = "failed"
			command.Progress = completed * 100 / len(command.Results)
			return
		case "timeout":
			command.Result = "timeout"
			command.Progress = completed * 100 / len(command.Results)
			return
		case "success":
			completed++
		}
	}
	command.Progress = completed * 100 / len(command.Results)
	if completed == len(command.Results) {
		command.Result = "success"
	} else {
		command.Result = "pending"
	}
}

func (store *Store) finishAlertLocked(deviceID, reason string, endedAt int64, duration int, maxConc float64) {
	alert := store.alerts[deviceID]
	if alert == nil {
		return
	}
	startedAt := store.alertStarted[deviceID]
	if duration <= 0 && endedAt >= startedAt {
		duration = int(endedAt - startedAt)
	}
	if maxConc <= 0 {
		maxConc = alert.Max
	}
	alert.Duration = duration
	alert.Max = maxConc
	alert.Status = "resolved"
	alert.EndReason = reason
	alert.EndedAt = time.Unix(endedAt, 0).Format("15:04:05")
	alert.StartedAtUnix = startedAt
	alert.EndedAtUnix = endedAt
	resolved := *alert
	store.alertHistory = append(store.alertHistory, resolved)
	go persistAlert(resolved)
	go deletePersistedAlert(resolved.ID)
	delete(store.alerts, deviceID)
	delete(store.alertStarted, deviceID)
	if device := store.devices[deviceID]; device != nil && device.device.Status == "alert" {
		device.device.Status = "normal"
	}
}

func (store *Store) hasActiveAlertsLocked(deviceID string) bool {
	return store.alerts[alertKey(deviceID)] != nil
}

func topicDeviceID(topic string) string {
	parts := strings.Split(strings.Trim(topic, "/"), "/")
	if len(parts) == 3 {
		return parts[2]
	}
	return ""
}
func timestamp(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return time.Now().Unix()
}

// maxDeviceClockSkew bounds how far a device-reported timestamp may drift from
// the server clock before it is treated as untrustworthy (e.g. an unset RTC).
const maxDeviceClockSkew = 5 * time.Minute

// sanitizeDeviceTimestamp falls back to the server receive time when a device
// reports an implausible clock, so a broken device RTC cannot make the device
// look permanently offline (time.Since(lastSeen) would never drop below the
// offline threshold) or corrupt alert/telemetry ordering.
func sanitizeDeviceTimestamp(deviceID string, reported int64) int64 {
	now := time.Now().Unix()
	if reported <= 0 {
		return now
	}
	skew := reported - now
	if skew < 0 {
		skew = -skew
	}
	if time.Duration(skew)*time.Second > maxDeviceClockSkew {
		logrus.Warnf("device %s reported implausible timestamp %d (skew %ds), using server time instead", deviceID, reported, skew)
		return now
	}
	return reported
}
func formatLastSeen(age time.Duration) string {
	if age < time.Second {
		return "刚刚"
	}
	return strconv.Itoa(int(age.Round(time.Second)/time.Second)) + "秒前"
}
