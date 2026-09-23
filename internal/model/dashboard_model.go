package model

import "time"

type DashboardSnapshot struct {
	UpdatedAt    time.Time           `json:"updated_at"`
	Summary      DashboardSummary    `json:"summary"`
	Devices      []DashboardDevice   `json:"devices"`
	Alerts       []DashboardAlert    `json:"alerts"`
	AlertHistory []DashboardAlert    `json:"alert_history"`
	Commands     []DashboardCommand  `json:"commands"`
	Trainings    []DashboardTraining `json:"trainings"`
}

type DashboardSummary struct {
	TotalDevices    int `json:"total_devices"`
	OnlineDevices   int `json:"online_devices"`
	AlertDevices    int `json:"alert_devices"`
	OfflineDevices  int `json:"offline_devices"`
	TrainingDevices int `json:"training_devices"`
}

type DashboardDevice struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Group         string  `json:"group"`
	Status        string  `json:"status"`
	Mode          string  `json:"mode"`
	Lat           float64 `json:"lat"`
	Lng           float64 `json:"lng"`
	PositionValid bool    `json:"position_valid"`
	Conc          float64 `json:"conc"`
	Threshold     float64 `json:"threshold"`
	Substance     string  `json:"substance"`
	Battery       int     `json:"battery"`
	RSSI          int     `json:"rssi"`
	Fall          bool    `json:"fall"`
	LastSeen      string  `json:"last_seen"`
}

type DashboardAlert struct {
	ID            string  `json:"id"`
	DeviceID      string  `json:"device_id"`
	DeviceName    string  `json:"device_name"`
	Group         string  `json:"group"`
	Substance     string  `json:"substance"`
	Fall          bool    `json:"fall"`
	StartedAt     string  `json:"started_at"`
	StartedAtUnix int64   `json:"started_at_unix"`
	Duration      int     `json:"duration"`
	Current       float64 `json:"current"`
	Max           float64 `json:"max"`
	Status        string  `json:"status"`
	Ack           string  `json:"ack"`
	EndReason     string  `json:"end_reason,omitempty"`
	EndedAt       string  `json:"ended_at,omitempty"`
	EndedAtUnix   int64   `json:"ended_at_unix,omitempty"`
}

type DashboardCommand struct {
	ID            string                   `json:"id"`
	Action        string                   `json:"action"`
	Target        string                   `json:"target"`
	Result        string                   `json:"result"`
	Progress      int                      `json:"progress"`
	CreatedAtUnix int64                    `json:"created_at_unix"`
	UpdatedAtUnix int64                    `json:"updated_at_unix"`
	TimeoutAtUnix int64                    `json:"timeout_at_unix,omitempty"`
	Results       []DashboardCommandResult `json:"results,omitempty"`
}

type DashboardCommandResult struct {
	DeviceID string `json:"device_id"`
	Result   string `json:"result"`
	Reason   string `json:"reason,omitempty"`
}

type DashboardTraining struct {
	ID        string                    `json:"id"`
	Name      string                    `json:"name"`
	Group     string                    `json:"group"`
	Devices   []string                  `json:"devices"`
	Mode      string                    `json:"mode"`
	Status    string                    `json:"status"`
	StartedAt string                    `json:"started_at"`
	EndedAt   string                    `json:"ended_at,omitempty"`
	Source    *DashboardPollutionSource `json:"source,omitempty"`
}

type DashboardPollutionSource struct {
	Substance string  `json:"substance"`
	Conc      float64 `json:"conc"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Radius    float64 `json:"radius"`
	Speed     float64 `json:"speed"`
}

type DashboardTelemetrySample struct {
	DeviceID  string  `json:"device_id"`
	Timestamp int64   `json:"timestamp"`
	Mode      string  `json:"mode"`
	Training  bool    `json:"training"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Conc      float64 `json:"conc"`
	Threshold float64 `json:"threshold"`
	Battery   int     `json:"battery"`
	RSSI      int     `json:"rssi"`
}

type DashboardGroup struct {
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Total    int    `json:"total"`
	Online   int    `json:"online"`
	Offline  int    `json:"offline"`
	Alerts   int    `json:"alerts"`
}

type DashboardAudit struct {
	ID        string         `json:"id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Target    string         `json:"target"`
	Result    string         `json:"result"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp int64          `json:"timestamp"`
}
