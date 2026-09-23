package router

import (
	"encoding/json"
	"net/http"
	"strconv"

	"data-server/internal/dashboard"
	"data-server/internal/model"

	"github.com/gin-gonic/gin"
)

func DashboardView() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "map/index.html", gin.H{})
	}
}

func DashboardSnapshot(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, store.Snapshot())
	}
}

func DashboardDevices(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"items": store.Snapshot().Devices})
	}
}

func DashboardGroups(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"items": store.Groups()})
	}
}

func RenameDevice(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := store.UpdateDeviceName(c.Param("device_id"), request.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func DeleteDevice(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.DeleteDevice(c.Param("device_id")); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func DashboardCommand(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			DeviceID string          `json:"device_id" binding:"required"`
			Type     int             `json:"type"`
			Message  json.RawMessage `json:"message" binding:"required"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if request.Type < 0 || request.Type > 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "notify type must be between 0 and 3"})
			return
		}
		var message any
		if err := json.Unmarshal(request.Message, &message); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "message must be valid JSON"})
			return
		}
		commandID, err := store.SendNotify(request.DeviceID, request.Type, message)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"id": commandID})
	}
}

func DashboardGroupCommand(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Type    int             `json:"type"`
			Message json.RawMessage `json:"message" binding:"required"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if request.Type < 0 || request.Type > 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group command type must be between 0 and 3"})
			return
		}
		var message any
		if err := json.Unmarshal(request.Message, &message); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "message must be valid JSON"})
			return
		}
		commandID, err := store.SendGroupNotify(c.Param("group"), request.Type, message)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"id": commandID})
	}
}

func StartTraining(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Name   string                          `json:"name" binding:"required"`
			Source *model.DashboardPollutionSource `json:"source"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		training, err := store.StartTrainingWithSource(c.Param("group"), request.Name, request.Source)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, training)
	}
}

func ExportAlertHistory(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "1000"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer"})
			return
		}
		from, err := parseUnixQuery(c.Query("from"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must be a Unix timestamp"})
			return
		}
		to, err := parseUnixQuery(c.Query("to"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to must be a Unix timestamp"})
			return
		}
		page := store.AlertHistory(c.Query("device_id"), c.Query("group"), from, to, "", 0, limit)
		if page.Items == nil && (limit <= 0 || limit > 1000) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 1000"})
			return
		}
		store.RecordAudit("system", "history.export", "dashboard_alert_history", "success", map[string]any{"count": len(page.Items)})
		c.Header("Content-Disposition", "attachment; filename=alert-history.json")
		c.JSON(http.StatusOK, gin.H{"items": page.Items})
	}
}

func EndTraining(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		training, err := store.EndTraining(c.Param("training_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, training)
	}
}

func UpdateTrainingSource(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var source model.DashboardPollutionSource
		if err := c.ShouldBindJSON(&source); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		training, err := store.UpdateTrainingSource(c.Param("training_id"), source)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, training)
	}
}

func TrainingHistory(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer"})
			return
		}
		history, err := store.TrainingHistory(limit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": history})
	}
}

func CommandHistory(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer"})
			return
		}
		history, err := store.CommandHistory(c.Query("result"), c.Query("target"), limit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": history})
	}
}

func TelemetryHistory(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer"})
			return
		}
		from, err := parseUnixQuery(c.Query("from"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must be a Unix timestamp"})
			return
		}
		to, err := parseUnixQuery(c.Query("to"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to must be a Unix timestamp"})
			return
		}
		items, err := store.TelemetryHistory(c.Query("device_id"), from, to, limit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func DeviceTrack(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "500"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer"})
			return
		}
		from, err := parseUnixQuery(c.Query("from"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must be a Unix timestamp"})
			return
		}
		to, err := parseUnixQuery(c.Query("to"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to must be a Unix timestamp"})
			return
		}
		items, err := store.TelemetryHistory(c.Param("device_id"), from, to, limit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"device_id": c.Param("device_id"), "items": items})
	}
}

func AuditHistory(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer"})
			return
		}
		items, err := store.AuditHistory(limit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func AssignDeviceGroup(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Group string `json:"group" binding:"required"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := store.AssignGroup(c.Param("device_id"), request.Group); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func ShareGroupPositions(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.ShareGroupPositions(c.Param("group")); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func ConfirmAlert(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.ConfirmAlert(c.Param("alert_id")); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func AlertHistory(store *dashboard.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be an integer"})
			return
		}
		from, err := parseUnixQuery(c.Query("from"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must be a Unix timestamp"})
			return
		}
		to, err := parseUnixQuery(c.Query("to"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to must be a Unix timestamp"})
			return
		}
		if from > 0 && to > 0 && from > to {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from must not be greater than to"})
			return
		}
		before, err := parseUnixQuery(c.Query("before"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "before must be a Unix timestamp"})
			return
		}
		page := store.AlertHistory(c.Query("device_id"), c.Query("group"), from, to, c.Query("before_id"), before, limit)
		if page.Items == nil && limit <= 0 || limit > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 1000"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": page.Items, "total": page.Total, "next_before": page.NextBefore, "next_before_id": page.NextBeforeID})
	}
}

func parseUnixQuery(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}
