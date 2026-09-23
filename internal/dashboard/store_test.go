package dashboard

import (
	"encoding/json"
	"testing"
	"time"

	"data-server/internal/model"
)

func TestAlertNormalEndUsesDeviceFinalValues(t *testing.T) {
	store := NewStore()
	store.applyEvent("device-1", eventEnvelope{Type: 0, Timestamp: 100, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{Names: []string{"DMMP"}, Conc: 3.8}}, 100)
	store.applyEvent("device-1", eventEnvelope{Type: 0, Timestamp: 102, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{Names: []string{"DMMP"}, Conc: 4.6}}, 102)
	store.applyEvent("device-1", eventEnvelope{Type: 1, Timestamp: 110, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{Duration: 100, MaxConc: 5.3}}, 110)

	snapshot := store.Snapshot()
	if len(snapshot.Alerts) != 0 || len(snapshot.AlertHistory) != 1 {
		t.Fatalf("expected one resolved alert, active=%d history=%d", len(snapshot.Alerts), len(snapshot.AlertHistory))
	}
	alert := snapshot.AlertHistory[0]
	if alert.Duration != 100 || alert.Max != 5.3 || alert.EndReason != "normal" {
		t.Fatalf("unexpected final alert: %+v", alert)
	}
}

func TestSubstanceAndFallShareOneAlert(t *testing.T) {
	store := NewStore()
	store.applyEvent("device-2", eventEnvelope{Type: 0, Timestamp: 200, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{Names: []string{"DMMP"}, Conc: 3.8}}, 200)
	store.applyEvent("device-2", eventEnvelope{Type: 0, Timestamp: 202, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{FallDetected: true}}, 202)

	snapshot := store.Snapshot()
	if len(snapshot.Alerts) != 1 || !snapshot.Alerts[0].Fall || snapshot.Alerts[0].Substance != "DMMP" {
		t.Fatalf("expected one combined alert: %+v", snapshot.Alerts)
	}
	store.applyEvent("device-2", eventEnvelope{Type: 1, Timestamp: 210, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{Names: []string{"DMMP"}, Duration: 10, MaxConc: 4.2}}, 210)
	snapshot = store.Snapshot()
	if len(snapshot.Alerts) != 0 || len(snapshot.AlertHistory) != 1 || !snapshot.AlertHistory[0].Fall {
		t.Fatalf("expected one resolved combined alert: active=%+v history=%+v", snapshot.Alerts, snapshot.AlertHistory)
	}
}

func TestOfflineEndsAlertAtLastTelemetry(t *testing.T) {
	store := NewStore()
	store.applyTelemetry("device-3", telemetryEnvelope{Timestamp: time.Now().Unix() - 16, Message: struct {
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
	}{DeviceID: "device-3"}})
	store.applyEvent("device-3", eventEnvelope{Type: 0, Timestamp: time.Now().Unix() - 20, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{Names: []string{"DMMP"}, Conc: 3.8}}, time.Now().Unix()-20)

	snapshot := store.Snapshot()
	if len(snapshot.Alerts) != 0 || len(snapshot.AlertHistory) != 1 || snapshot.AlertHistory[0].EndReason != "offline" {
		t.Fatalf("expected offline-resolved alert: %+v", snapshot.AlertHistory)
	}
}

func TestValidateNotifyMessage(t *testing.T) {
	tests := []struct {
		name       string
		notifyType int
		message    any
		wantErr    bool
	}{
		{name: "alarm notification", notifyType: 0, message: "警报", wantErr: false},
		{name: "normal notification", notifyType: 0, message: "正常", wantErr: false},
		{name: "evacuate notification", notifyType: 0, message: "撤离", wantErr: false},
		{name: "empty notification", notifyType: 0, message: " ", wantErr: true},
		{name: "notification must be text", notifyType: 0, message: float64(1), wantErr: true},
		{name: "valid action", notifyType: 1, message: map[string]any{"action": "evacuate"}, wantErr: false},
		{name: "invalid action", notifyType: 1, message: map[string]any{"action": "stop"}, wantErr: true},
		{name: "valid pollution source", notifyType: 1, message: map[string]any{"action": "pollution_source", "substance": "DMMP", "conc": 5.0}, wantErr: false},
		{name: "pollution source missing substance", notifyType: 1, message: map[string]any{"action": "pollution_source", "conc": 5.0}, wantErr: true},
		{name: "pollution source non-positive conc", notifyType: 1, message: map[string]any{"action": "pollution_source", "substance": "DMMP", "conc": 0.0}, wantErr: true},
		{name: "valid pollution source clear", notifyType: 1, message: map[string]any{"action": "pollution_source_clear"}, wantErr: false},
		{name: "valid mode", notifyType: 3, message: map[string]any{"mode": "training"}, wantErr: false},
		{name: "invalid mode", notifyType: 3, message: map[string]any{"mode": "exercise"}, wantErr: true},
		{name: "valid position list", notifyType: 2, message: map[string]any{"devices": []map[string]any{{"device_id": "d1", "lat": 1.0, "lng": 2.0}}}, wantErr: false},
		{name: "unsupported type", notifyType: 4, message: float64(1), wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateNotifyMessage(test.notifyType, test.message); (err != nil) != test.wantErr {
				t.Fatalf("validateNotifyMessage() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestTelemetryRejectsObjectAlarmNames(t *testing.T) {
	payload := []byte(`{"message":{"gnss":{"fixed":false,"lat":22.6141186,"speed":0,"lng":113.836998},"device_id":"864793080139046","timestamp":946685775,"sensor":{"pid":{"conc":0,"alarm":false},"battery":{"pct":95,"voltage":8.3872185},"alarm":{"names":{}}},"rssi":-57,"gsensor":{"magnitude":1.4884094,"fall_detected":false},"mode":"monitor"},"timestamp":946685775}`)
	var envelope telemetryEnvelope
	if err := json.Unmarshal(payload, &envelope); err == nil {
		t.Fatal("expected protocol violation for object-shaped alarm names")
	}
}

func TestTelemetryWithLegacyClockStillOnboardsDevice(t *testing.T) {
	payload := []byte(`{"message":{"gnss":{"fixed":false,"lat":22.6141186,"speed":0,"lng":113.836998},"device_id":"864793080139046","timestamp":946684914,"sensor":{"pid":{"conc":0,"alarm":false,"threshold":50},"battery":{"pct":95,"voltage":8.3718004},"alarm":{"names":[]}},"rssi":-60,"gsensor":{"magnitude":1.5005593,"fall_detected":false},"mode":"monitor"},"timestamp":946684914}`)
	var envelope telemetryEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode telemetry: %v", err)
	}
	if err := validateTelemetry(envelope); err != nil {
		t.Fatalf("validate telemetry: %v", err)
	}

	store := NewStore()
	store.applyTelemetry(envelope.Message.DeviceID, envelope)
	snapshot := store.Snapshot()
	if len(snapshot.Devices) != 1 || snapshot.Devices[0].ID != "864793080139046" {
		t.Fatalf("expected device in snapshot, got %+v", snapshot.Devices)
	}
	if snapshot.Devices[0].Status == "offline" {
		t.Fatalf("expected device to be online despite legacy device clock, got %+v", snapshot.Devices[0])
	}
}

func TestTrainingClosesLoopOnCommandAck(t *testing.T) {
	store := NewStore()
	store.trainings["t1"] = &model.DashboardTraining{ID: "t1", Mode: "training", Status: "starting"}
	store.trainingCommands["cmd1"] = "t1"
	store.commands["cmd1"] = &model.DashboardCommand{ID: "cmd1", Results: []model.DashboardCommandResult{{DeviceID: "device-1", Result: "pending"}}}

	store.applyEvent("device-1", eventEnvelope{Type: 2, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{ID: "cmd1", Result: "success"}}, time.Now().Unix())

	if status := store.trainings["t1"].Status; status != "active" {
		t.Fatalf("expected training to become active after ACK, got %q", status)
	}
	if _, stillTracked := store.trainingCommands["cmd1"]; stillTracked {
		t.Fatal("expected command mapping to be cleared after resolution")
	}
}

func TestTrainingEndsOnFailedCommandAck(t *testing.T) {
	store := NewStore()
	store.trainings["t2"] = &model.DashboardTraining{ID: "t2", Mode: "training", Status: "starting"}
	store.trainingCommands["cmd2"] = "t2"
	store.commands["cmd2"] = &model.DashboardCommand{ID: "cmd2", Results: []model.DashboardCommandResult{{DeviceID: "device-2", Result: "pending"}}}

	store.applyEvent("device-2", eventEnvelope{Type: 2, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"peak_conc"`
	}{ID: "cmd2", Result: "failed"}}, time.Now().Unix())

	if status := store.trainings["t2"].Status; status != "ended" {
		t.Fatalf("expected training to end after failed ACK, got %q", status)
	}
}

func TestGroupsOnlyContainAssignedDevices(t *testing.T) {
	store := NewStore()
	if groups := store.Groups(); len(groups) != 0 {
		t.Fatalf("expected no default groups, got %+v", groups)
	}

	store.devices["device-1"] = &deviceState{device: model.DashboardDevice{ID: "device-1", Group: "未分组", Status: "normal"}}
	store.devices["device-2"] = &deviceState{device: model.DashboardDevice{ID: "device-2", Group: "编队A", Status: "normal"}}
	groups := store.Groups()
	if len(groups) != 1 || groups[0].Name != "编队A" || groups[0].Total != 1 {
		t.Fatalf("expected only assigned group, got %+v", groups)
	}
}

func TestCollectPositionSharesUsesOnlinePeersOnly(t *testing.T) {
	store := NewStore()
	now := time.Unix(100, 0)
	store.devices["A"] = &deviceState{device: model.DashboardDevice{ID: "A", Group: "编队01", Lat: 22.1, Lng: 114.1, PositionValid: true}, lastSeen: now}
	store.devices["B"] = &deviceState{device: model.DashboardDevice{ID: "B", Group: "编队01", Lat: 22.2, Lng: 114.2, PositionValid: true}, lastSeen: now}
	store.devices["C"] = &deviceState{device: model.DashboardDevice{ID: "C", Group: "编队02", Lat: 22.3, Lng: 114.3, PositionValid: true}, lastSeen: now}

	store.mu.Lock()
	shares := store.collectPositionSharesLocked(now.Add(5 * time.Second))
	store.mu.Unlock()
	if len(shares) != 2 {
		t.Fatalf("expected two same-group shares, got %d", len(shares))
	}
	for _, share := range shares {
		var payload struct {
			Type    int `json:"type"`
			Message struct {
				Devices []struct {
					ID string `json:"device_id"`
				} `json:"devices"`
			} `json:"message"`
		}
		if err := json.Unmarshal(share.data, &payload); err != nil {
			t.Fatalf("decode position share: %v", err)
		}
		if payload.Type != 2 || len(payload.Message.Devices) != 1 || payload.Message.Devices[0].ID == share.target || payload.Message.Devices[0].ID == "C" {
			t.Fatalf("invalid share for %s: %+v", share.target, payload)
		}
	}
}
