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
		MaxConc      float64  `json:"max_conc"`
	}{Names: []string{"DMMP"}, Conc: 3.8}}, 100)
	store.applyEvent("device-1", eventEnvelope{Type: 0, Timestamp: 102, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"max_conc"`
	}{Names: []string{"DMMP"}, Conc: 4.6}}, 102)
	store.applyEvent("device-1", eventEnvelope{Type: 1, Timestamp: 110, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"max_conc"`
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
		MaxConc      float64  `json:"max_conc"`
	}{Names: []string{"DMMP"}, Conc: 3.8}}, 200)
	store.applyEvent("device-2", eventEnvelope{Type: 0, Timestamp: 202, Message: struct {
		Names        []string `json:"names"`
		Conc         float64  `json:"conc"`
		FallDetected bool     `json:"fall_detected"`
		ID           string   `json:"id"`
		Result       string   `json:"result"`
		Reason       string   `json:"reason"`
		Duration     int      `json:"duration"`
		MaxConc      float64  `json:"max_conc"`
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
		MaxConc      float64  `json:"max_conc"`
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
		MaxConc      float64  `json:"max_conc"`
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
		{name: "alarm notification", notifyType: 0, message: float64(1), wantErr: false},
		{name: "normal notification", notifyType: 0, message: float64(2), wantErr: false},
		{name: "evacuate notification", notifyType: 0, message: float64(3), wantErr: false},
		{name: "invalid notification code", notifyType: 0, message: float64(4), wantErr: true},
		{name: "notification must be numeric", notifyType: 0, message: "集合", wantErr: true},
		{name: "valid action", notifyType: 1, message: map[string]any{"action": "evacuate"}, wantErr: false},
		{name: "invalid action", notifyType: 1, message: map[string]any{"action": "stop"}, wantErr: true},
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

func TestCollectPositionSharesUsesOnlinePeersOnly(t *testing.T) {
	store := NewStore()
	now := time.Unix(100, 0)
	store.devices["A"] = &deviceState{device: model.DashboardDevice{ID: "A", Group: "编队01", Lat: 22.1, Lng: 114.1}, lastSeen: now}
	store.devices["B"] = &deviceState{device: model.DashboardDevice{ID: "B", Group: "编队01", Lat: 22.2, Lng: 114.2}, lastSeen: now}
	store.devices["C"] = &deviceState{device: model.DashboardDevice{ID: "C", Group: "编队02", Lat: 22.3, Lng: 114.3}, lastSeen: now}

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
