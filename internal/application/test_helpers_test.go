package application

import (
	"context"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/repository/memory"
)

type fixedClock struct{ value time.Time }

func (f fixedClock) Now() time.Time { return f.value }

type fixture struct {
	store      *memory.Store
	clock      fixedClock
	lab        domain.Laboratory
	instrument domain.Instrument
	item       domain.CalibrationItem
	plan       domain.CalibrationPlan
}

func newFixture(t interface{ Fatal(...any) }) fixture {
	ctx := context.Background()
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	lab, err := domain.NewLaboratory("LAB-A", "Analytical Lab", "A-101", "manager", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	instrument, err := domain.NewInstrument(domain.InstrumentInput{LaboratoryID: lab.ID, AssetNumber: "LAB-001", Name: "Pressure Gauge", Model: "PG-10", OwnerID: "owner", Criticality: domain.CriticalityCritical}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateInstrument(ctx, instrument); err != nil {
		t.Fatal(err)
	}
	standard, err := domain.NewInstrument(domain.InstrumentInput{LaboratoryID: lab.ID, AssetNumber: "STD-001", Name: "Pressure Standard", Model: "REF-100", SerialNumber: "RS-100", OwnerID: "metrology", Criticality: domain.CriticalityCritical}, now)
	if err != nil {
		t.Fatal(err)
	}
	standard.ID = "standard"
	standard.Status = domain.StatusQualified
	standard.NextDueAt = now.AddDate(1, 0, 0)
	if err := store.CreateInstrument(ctx, standard); err != nil {
		t.Fatal(err)
	}
	item, err := domain.NewCalibrationItem("PRESS", "Pressure accuracy", 180, 30, 0.5, "kPa", "standard", []string{"PG-10"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateItem(ctx, item); err != nil {
		t.Fatal(err)
	}
	plan, err := domain.NewPlan(instrument.ID, item.ID, now.Add(24*time.Hour), "operator", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	return fixture{store: store, clock: fixedClock{now}, lab: lab, instrument: instrument, item: item, plan: plan}
}
