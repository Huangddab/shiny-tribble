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
	if got := simulatedConcentration(source, 0.0009, 0, 20*time.Second); math.Abs(got-1.25) > 0.03 {
		t.Fatalf("at half maximum radius: got %v", got)
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
