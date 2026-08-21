package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/platform/clock"
	"github.com/wyw14/cry-055/internal/repository/memory"
)

func TestCreateCalibrationItemValidatesApplicabilityAndRollsBackRejectedStandard(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	store := memory.New()
	lab, err := domain.NewLaboratory("LAB-CAL", "Calibration", "C-101", "manager", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	newStandard := func(asset, model, serial string, status domain.InstrumentStatus) domain.Instrument {
		instrument, err := domain.NewInstrument(domain.InstrumentInput{
			LaboratoryID: lab.ID, AssetNumber: asset, Name: asset,
			Model: model, SerialNumber: serial, OwnerID: "metrology",
			Criticality: domain.CriticalityCritical,
		}, now)
		if err != nil {
			t.Fatal(err)
		}
		instrument.Status = status
		instrument.NextDueAt = now.AddDate(1, 0, 0)
		if err := store.CreateInstrument(ctx, instrument); err != nil {
			t.Fatal(err)
		}
		return instrument
	}
	rejectedStandard := newStandard("STD-LOCKED", "REF-50", "RS-050", domain.StatusDisabled)
	qualifiedStandard := newStandard("STD-VALID", "REF-100", "RS-100", domain.StatusQualified)
	service := NewCatalogService(store, store, store, store, clock.Fixed{Value: now})

	_, err = service.CreateCalibrationItem(ctx, CalibrationItemInput{
		Code: "TEMP", Name: "Temperature verification", PeriodDays: 90, WarningDays: 10,
		Tolerance: 0.2, Unit: "C", ReferenceStandardID: rejectedStandard.ID,
		ApplicableModels: []string{"TP-20"},
	})
	var validation domain.ValidationError
	if !errors.As(err, &validation) {
		t.Errorf("expected rejected standard validation error, got %v", err)
	}
	if _, lookupErr := store.GetItemByCode(ctx, "TEMP"); !errors.Is(lookupErr, domain.ErrNotFound) {
		t.Errorf("rejected item must roll back, lookup error=%v", lookupErr)
	}

	item, err := service.CreateCalibrationItem(ctx, CalibrationItemInput{
		Code: "PRESS", Name: "Pressure accuracy", PeriodDays: 180, WarningDays: 30,
		Tolerance: 0.5, Unit: "kPa", ReferenceStandardID: qualifiedStandard.ID,
		ApplicableModels: []string{" pg-10 ", "PG-10", "tp-20"},
	})
	if err != nil {
		t.Fatalf("create valid calibration item: %v", err)
	}
	if want := []string{"PG-10", "TP-20"}; !reflect.DeepEqual(item.ApplicableModels, want) {
		t.Errorf("applicable models=%v, want %v", item.ApplicableModels, want)
	}
	if !item.AppliesTo("  pg-10 ") || item.AppliesTo("humidity-9") {
		t.Errorf("canonical applicability mismatch for %v", item.ApplicableModels)
	}
	persisted, err := store.GetItemByCode(ctx, "press")
	if err != nil || persisted.ID != item.ID || !reflect.DeepEqual(persisted.ApplicableModels, item.ApplicableModels) {
		t.Errorf("persisted item mismatch: item=%+v err=%v", persisted, err)
	}
}
