package dashboard

import (
	"math"
	"testing"
	"time"

	"data-server/internal/model"
)

func TestSimulatedConcentrationByDistanceAndTime(t *testing.T) {
	source := model.DashboardPollutionSource{Substance: "DMMP", Conc: 5, Lat: 0, Lng: 0, Radius: 200, Speed: 10}
	if got := simulatedConcentration(source, 0, 0, 0); got != 0 {
		t.Fatalf("before expansion: got %v", got)
	}
	if got := simulatedConcentration(source, 0, 0, 10*time.Second); got != 5 {
		t.Fatalf("at source: got %v", got)
	}
	// 0.0009 degrees of latitude is approximately 100 metres.
	if got := simulatedConcentration(source, 0.0009, 0, 5*time.Second); got != 0 {
		t.Fatalf("outside current radius: got %v", got)
	}
	if got := simulatedConcentration(source, 0.0009, 0, 20*time.Second); got <= 0 || got >= source.Conc {
		t.Fatalf("at half maximum radius: got %v", got)
	}
	first := simulatedConcentration(source, 0, 0, 20*time.Second)
	later := simulatedConcentration(source, 0, 0, 30*time.Second)
	if math.Abs(first-later) < 0.01 {
		t.Fatalf("concentration should fluctuate over time: %v then %v", first, later)
	}
	if sourceWave(0, 0) != 1 || math.Abs(sourceWave(20*time.Second, 0)-sourceWave(30*time.Second, 0)) < 0.01 {
		t.Fatal("source wave must start at the configured concentration and evolve without GNSS")
	}
}

func TestValidateTrainingSource(t *testing.T) {
	source := model.DashboardPollutionSource{Substance: "DMMP", Conc: 5, Lat: 22, Lng: 113, Radius: 200, Speed: 2}
	if err := validateTrainingSource(&source); err != nil {
		t.Fatal(err)
	}
	source.Speed = math.Inf(1)
	if err := validateTrainingSource(&source); err == nil {
		t.Fatal("infinite speed must be rejected")
	}
}

func TestGCJ02SourceMatchesWGS84Device(t *testing.T) {
	lat, lng := 22.6335, 113.9035
	mapLat, mapLng := wgs84ToGCJ02(lat, lng)
	if math.Abs(mapLat-lat) < 0.001 || math.Abs(mapLng-lng) < 0.001 {
		t.Fatal("expected a meaningful map offset for mainland GPS coordinates")
	}
	source := model.DashboardPollutionSource{Conc: 5, Lat: mapLat, Lng: mapLng, Radius: 100, Speed: 10}
	if got := simulatedConcentration(source, lat, lng, 10*time.Second); got != 5 {
		t.Fatalf("device at the map source must receive full concentration, got %v", got)
	}
	if got := simulatedConcentration(source, lat+0.002, lng, 10*time.Second); got != 0 {
		t.Fatalf("device outside radius must receive zero, got %v", got)
	}
	outsideLat, outsideLng := wgs84ToGCJ02(35.6895, 139.6917)
	if outsideLat != 35.6895 || outsideLng != 139.6917 {
		t.Fatal("coordinates outside mainland China must remain unchanged")
	}
}
