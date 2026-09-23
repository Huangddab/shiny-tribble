package dashboard

import (
	"fmt"
	"math"
	"strings"
	"time"

	"data-server/internal/model"

	"github.com/sirupsen/logrus"
)

// This is a deterministic exercise curve, not an atmospheric dispersion model.
type sourceDelivery struct {
	conc float64
	at   time.Time
}

type sourceUpdate struct {
	trainingID, deviceID, substance string
	conc float64
	clear bool
}

func validateTrainingSource(source *model.DashboardPollutionSource) error {
	if source == nil {
		return nil
	}
	if strings.TrimSpace(source.Substance) == "" || !finitePositive(source.Conc) ||
		!finitePositive(source.Radius) || !finitePositive(source.Speed) ||
		math.IsNaN(source.Lat) || math.IsNaN(source.Lng) ||
		math.Abs(source.Lat) > 90 || math.Abs(source.Lng) > 180 {
		return fmt.Errorf("pollution source requires substance, positive conc/radius/speed and valid coordinates")
	}
	return nil
}

func finitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

func sourceDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000.0
	toRad := math.Pi / 180
	dLat, dLng := (lat2-lat1)*toRad, (lng2-lng1)*toRad
	a := math.Pow(math.Sin(dLat/2), 2) +
		math.Cos(lat1*toRad)*math.Cos(lat2*toRad)*math.Pow(math.Sin(dLng/2), 2)
	return 2 * earthRadius * math.Asin(math.Sqrt(math.Min(1, a)))
}

func simulatedConcentration(source model.DashboardPollutionSource, lat, lng float64, elapsed time.Duration) float64 {
	radius := math.Min(source.Radius, source.Speed*elapsed.Seconds())
	if radius <= 0 {
		return 0
	}
	distance := sourceDistance(source.Lat, source.Lng, lat, lng)
	if distance >= radius {
		return 0
	}
	factor := 1 - distance/radius
	return math.Round(source.Conc*factor*factor*100) / 100
}

// UpdateTrainingSource changes the exercise parameters; the next tick delivers
// a separately calculated concentration to each positioned device.
func (store *Store) UpdateTrainingSource(trainingID string, source model.DashboardPollutionSource) (model.DashboardTraining, error) {
	if err := validateTrainingSource(&source); err != nil {
		return model.DashboardTraining{}, err
	}
	store.mu.Lock()
	training := store.trainings[trainingID]
	if training == nil || training.Status == "ended" || training.Source == nil {
		store.mu.Unlock()
		return model.DashboardTraining{}, fmt.Errorf("training not active or has no pollution source")
	}
	training.Source = &source
	result := *training
	for _, deviceID := range training.Devices {
		delete(store.sourceSent, trainingID+":"+deviceID)
	}
	store.mu.Unlock()
	go persistTraining(result)
	go persistAudit("system", "training.source_update", trainingID, "accepted", map[string]any{"source": source})
	return result, nil
}

func (store *Store) updateTrainingSources(now time.Time) {
	store.mu.RLock()
	var updates []sourceUpdate
	for id, training := range store.trainings {
		if training.Status != "active" || training.Source == nil {
			continue
		}
		started := store.trainingStarted[id]
		if started.IsZero() { // Restored training: start a new expansion clock.
			started = now
		}
		for _, deviceID := range training.Devices {
			device := store.devices[deviceID]
			if device == nil || !device.device.PositionValid || now.Sub(device.lastSeen) >= 15*time.Second {
				continue
			}
			conc := simulatedConcentration(*training.Source, device.device.Lat, device.device.Lng, now.Sub(started))
			key := id + ":" + deviceID
			previous, sent := store.sourceSent[key]
			if conc == 0 && (!sent || previous.conc == 0) {
				continue
			}
			if sent && previous.conc == conc && now.Sub(previous.at) < 30*time.Second {
				continue
			}
			if sent && now.Sub(previous.at) < 10*time.Second {
				continue
			}
			updates = append(updates, sourceUpdate{id, deviceID, training.Source.Substance, conc, conc == 0})
		}
	}
	store.mu.RUnlock()
	for _, update := range updates {
		store.mu.RLock()
		training := store.trainings[update.trainingID]
		active := training != nil && training.Status == "active" && training.Source != nil
		store.mu.RUnlock()
		if !active {
			continue
		}
		message := map[string]any{"action": "pollution_source", "substance": update.substance, "conc": update.conc}
		if update.clear {
			message = map[string]any{"action": "pollution_source_clear"}
		}
		if _, err := store.SendNotify(update.deviceID, 1, message); err != nil {
			logrus.Warnf("training %s source update for %s failed: %v", update.trainingID, update.deviceID, err)
			continue
		}
		store.mu.Lock()
		store.sourceSent[update.trainingID+":"+update.deviceID] = sourceDelivery{update.conc, now}
		store.mu.Unlock()
	}
}
