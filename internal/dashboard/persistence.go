package dashboard

import (
	"data-server/internal/database"
	"data-server/internal/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func persistDocument(collection string, document any) {
	if database.GetDatabase() == nil {
		return
	}
	if _, err := database.InsertOne(collection, document); err != nil {
		logrus.Warnf("persist %s failed: %v", collection, err)
	}
}

func persistCommand(command model.DashboardCommand, timestamp int64) {
	results := make([]commandResultRecord, 0, len(command.Results))
	for _, result := range command.Results {
		results = append(results, commandResultRecord{DeviceID: result.DeviceID, Result: result.Result, Reason: result.Reason})
	}
	document := modelCommandRecord{ID: command.ID, Action: command.Action, Target: command.Target, Result: command.Result, Progress: command.Progress, Results: results, CreatedAt: command.CreatedAtUnix, UpdatedAt: timestamp, TimeoutAt: command.TimeoutAtUnix}
	if database.GetDatabase() == nil {
		return
	}
	if _, err := database.UpdateOne("dashboard_commands", bson.M{"_id": command.ID}, bson.M{"$set": document, "$setOnInsert": bson.M{"_id": command.ID, "created_at": timestamp}}, options.Update().SetUpsert(true)); err != nil {
		logrus.Warnf("persist dashboard_commands failed: %v", err)
	}
}

func loadCommandHistory(result, target string, limit int) ([]model.DashboardCommand, error) {
	if database.GetDatabase() == nil {
		return nil, errors.New("database not initialized")
	}
	filter := bson.M{}
	if result != "" {
		filter["result"] = result
	}
	if target != "" {
		filter["target"] = bson.M{"$regex": target}
	}
	var records []modelCommandRecord
	if err := database.FindAll("dashboard_commands", filter, &records, options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetLimit(int64(limit))); err != nil {
		return nil, err
	}
	commands := make([]model.DashboardCommand, 0, len(records))
	for _, record := range records {
		results := make([]model.DashboardCommandResult, 0, len(record.Results))
		for _, item := range record.Results {
			results = append(results, model.DashboardCommandResult{DeviceID: item.DeviceID, Result: item.Result, Reason: item.Reason})
		}
		commands = append(commands, model.DashboardCommand{ID: record.ID, Action: record.Action, Target: record.Target, Result: record.Result, Progress: record.Progress, CreatedAtUnix: record.CreatedAt, UpdatedAtUnix: record.UpdatedAt, TimeoutAtUnix: record.TimeoutAt, Results: results})
	}
	return commands, nil
}

func persistAlert(alert model.DashboardAlert) {
	persistDocument("dashboard_alert_history", alertDocument(alert))
}

func cleanupLegacyDashboardData() {
	if database.GetDatabase() == nil {
		return
	}
	filter := bson.M{"ended_at_unix": bson.M{"$exists": false}}
	result, err := database.DeleteMany("dashboard_alert_history", filter)
	if err != nil {
		logrus.Warnf("cleanup legacy dashboard_alert_history failed: %v", err)
		return
	}
	if result.DeletedCount > 0 {
		logrus.Infof("deleted %d legacy dashboard alert history records", result.DeletedCount)
	}
}

func persistTraining(training model.DashboardTraining) {
	if database.GetDatabase() == nil {
		return
	}
	document := bson.M{"name": training.Name, "group": training.Group, "devices": training.Devices, "mode": training.Mode, "status": training.Status, "started_at": training.StartedAt, "ended_at": training.EndedAt, "source": training.Source}
	if _, err := database.UpdateOne("dashboard_trainings", bson.M{"_id": training.ID}, bson.M{"$set": document, "$setOnInsert": bson.M{"_id": training.ID}}, options.Update().SetUpsert(true)); err != nil {
		logrus.Warnf("persist dashboard_trainings failed: %v", err)
	}
}

func persistDevice(device model.DashboardDevice) {
	if database.GetDatabase() == nil {
		return
	}
	document := bson.M{"name": device.Name, "group": device.Group, "status": device.Status, "mode": device.Mode}
	if _, err := database.UpdateOne("dashboard_devices", bson.M{"_id": device.ID}, bson.M{"$set": document, "$setOnInsert": bson.M{"_id": device.ID}}, options.Update().SetUpsert(true)); err != nil {
		logrus.Warnf("persist dashboard_devices failed: %v", err)
	}
}

func persistTelemetrySample(sample model.DashboardTelemetrySample) {
	persistDocument("dashboard_telemetry_samples", sample)
}

func loadTelemetryHistory(deviceID string, from, to int64, limit int) ([]model.DashboardTelemetrySample, error) {
	if database.GetDatabase() == nil {
		return nil, errors.New("database not initialized")
	}
	filter := bson.M{}
	if deviceID != "" {
		filter["device_id"] = deviceID
	}
	timestampFilter := bson.M{}
	if from > 0 {
		timestampFilter["$gte"] = from
	}
	if to > 0 {
		timestampFilter["$lte"] = to
	}
	if len(timestampFilter) > 0 {
		filter["timestamp"] = timestampFilter
	}
	var samples []model.DashboardTelemetrySample
	if err := database.FindAll("dashboard_telemetry_samples", filter, &samples, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(limit))); err != nil {
		return nil, err
	}
	return samples, nil
}

func persistAudit(actor, action, target, result string, metadata map[string]any) {
	entry := auditRecord{ID: uuid.NewString(), Actor: actor, Action: action, Target: target, Result: result, Metadata: metadata, Timestamp: time.Now().Unix()}
	persistDocument("dashboard_audit_logs", entry)
}

func loadAuditHistory(limit int) ([]model.DashboardAudit, error) {
	if database.GetDatabase() == nil {
		return nil, errors.New("database not initialized")
	}
	var records []auditRecord
	if err := database.FindAll("dashboard_audit_logs", bson.M{}, &records, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(limit))); err != nil {
		return nil, err
	}
	items := make([]model.DashboardAudit, 0, len(records))
	for _, record := range records {
		items = append(items, model.DashboardAudit{ID: record.ID, Actor: record.Actor, Action: record.Action, Target: record.Target, Result: record.Result, Metadata: record.Metadata, Timestamp: record.Timestamp})
	}
	return items, nil
}

func loadTrainingHistory(limit int) ([]model.DashboardTraining, error) {
	if database.GetDatabase() == nil {
		return nil, errors.New("database not initialized")
	}
	var records []trainingRecord
	if err := database.FindAll("dashboard_trainings", bson.M{}, &records, options.Find().SetSort(bson.D{{Key: "started_at", Value: -1}}).SetLimit(int64(limit))); err != nil {
		return nil, err
	}
	trainings := make([]model.DashboardTraining, 0, len(records))
	for _, record := range records {
		trainings = append(trainings, model.DashboardTraining{ID: record.ID, Name: record.Name, Group: record.Group, Devices: record.Devices, Mode: record.Mode, Status: record.Status, StartedAt: record.StartedAt, EndedAt: record.EndedAt, Source: record.Source})
	}
	return trainings, nil
}

func persistAlertConfirmation(alert model.DashboardAlert) {
	if database.GetDatabase() == nil {
		return
	}
	if _, err := database.UpdateOne("dashboard_alerts", bson.M{"_id": alert.ID}, bson.M{"$set": alertFields(alert), "$setOnInsert": bson.M{"_id": alert.ID}}, options.Update().SetUpsert(true)); err != nil {
		logrus.Warnf("persist dashboard_alerts failed: %v", err)
	}
}

func deletePersistedAlert(alertID string) {
	if database.GetDatabase() == nil {
		return
	}
	if _, err := database.DeleteOne("dashboard_alerts", bson.M{"_id": alertID}); err != nil {
		logrus.Warnf("delete dashboard_alerts failed: %v", err)
	}
}

func loadAlertHistory(deviceID, group string, from, to int64, beforeID string, before int64, limit int) ([]model.DashboardAlert, error) {
	if database.GetDatabase() == nil {
		return nil, errors.New("database not initialized")
	}
	filter := bson.M{}
	if deviceID != "" {
		filter["device"] = deviceID
	}
	if group != "" {
		filter["group"] = group
	}
	conditions := bson.A{bson.M{"ended_at_unix": bson.M{"$exists": true}}}
	if from > 0 || to > 0 {
		endedAt := bson.M{}
		if from > 0 {
			endedAt["$gte"] = from
		}
		if to > 0 {
			endedAt["$lte"] = to
		}
		conditions = append(conditions, bson.M{"ended_at_unix": endedAt})
	}
	if before > 0 {
		if beforeID == "" {
			conditions = append(conditions, bson.M{"ended_at_unix": bson.M{"$lt": before}})
		} else {
			conditions = append(conditions, bson.M{"$or": bson.A{
				bson.M{"ended_at_unix": bson.M{"$lt": before}},
				bson.M{"ended_at_unix": before, "_id": bson.M{"$lt": beforeID}},
			}})
		}
	}
	filter["$and"] = conditions
	var records []alertHistoryRecord
	if err := database.FindAll("dashboard_alert_history", filter, &records, options.Find().SetSort(bson.D{{Key: "ended_at_unix", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(int64(limit))); err != nil {
		return nil, err
	}
	alerts := make([]model.DashboardAlert, 0, len(records))
	for _, record := range records {
		alerts = append(alerts, model.DashboardAlert{ID: record.ID, Device: record.Device, Group: record.Group, Substance: record.Substance, Fall: record.Fall, StartedAt: record.StartedAt, StartedAtUnix: record.StartedAtUnix, Duration: record.Duration, Current: record.Current, Max: record.Max, Status: record.Status, Ack: record.Ack, EndReason: record.EndReason, EndedAt: record.EndedAt, EndedAtUnix: record.EndedAtUnix})
	}
	return alerts, nil
}

func cloneDashboardCommand(command model.DashboardCommand) model.DashboardCommand {
	command.Results = append([]model.DashboardCommandResult(nil), command.Results...)
	return command
}

type modelCommandRecord struct {
	ID        string                `bson:"-"`
	Action    string                `bson:"action"`
	Target    string                `bson:"target"`
	Result    string                `bson:"result"`
	Progress  int                   `bson:"progress"`
	Results   []commandResultRecord `bson:"results"`
	CreatedAt int64                 `bson:"created_at"`
	UpdatedAt int64                 `bson:"updated_at"`
	TimeoutAt int64                 `bson:"timeout_at,omitempty"`
}

type commandResultRecord struct {
	DeviceID string `bson:"device_id"`
	Result   string `bson:"result"`
	Reason   string `bson:"reason,omitempty"`
}

type trainingRecord struct {
	ID        string                          `bson:"_id"`
	Name      string                          `bson:"name"`
	Group     string                          `bson:"group"`
	Devices   []string                        `bson:"devices"`
	Mode      string                          `bson:"mode"`
	Status    string                          `bson:"status"`
	StartedAt string                          `bson:"started_at"`
	EndedAt   string                          `bson:"ended_at"`
	Source    *model.DashboardPollutionSource `bson:"source,omitempty"`
}

type auditRecord struct {
	ID        string         `bson:"_id"`
	Actor     string         `bson:"actor"`
	Action    string         `bson:"action"`
	Target    string         `bson:"target"`
	Result    string         `bson:"result"`
	Metadata  map[string]any `bson:"metadata,omitempty"`
	Timestamp int64          `bson:"timestamp"`
}

type alertHistoryRecord struct {
	ID            string  `bson:"_id"`
	Device        string  `bson:"device"`
	Group         string  `bson:"group"`
	Substance     string  `bson:"substance"`
	Fall          bool    `bson:"fall"`
	StartedAt     string  `bson:"started_at"`
	Duration      int     `bson:"duration"`
	Current       float64 `bson:"current"`
	Max           float64 `bson:"max"`
	Status        string  `bson:"status"`
	Ack           string  `bson:"ack"`
	EndReason     string  `bson:"end_reason"`
	EndedAt       string  `bson:"ended_at"`
	StartedAtUnix int64   `bson:"started_at_unix"`
	EndedAtUnix   int64   `bson:"ended_at_unix"`
}

func alertDocument(alert model.DashboardAlert) bson.M {
	document := alertFields(alert)
	document["_id"] = alert.ID
	return document
}

func alertFields(alert model.DashboardAlert) bson.M {
	return bson.M{"device": alert.Device, "group": alert.Group, "substance": alert.Substance, "fall": alert.Fall, "started_at": alert.StartedAt, "started_at_unix": alert.StartedAtUnix, "duration": alert.Duration, "current": alert.Current, "max": alert.Max, "status": alert.Status, "ack": alert.Ack, "end_reason": alert.EndReason, "ended_at": alert.EndedAt, "ended_at_unix": alert.EndedAtUnix}
}
