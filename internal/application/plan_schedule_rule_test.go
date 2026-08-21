package application

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-055/internal/domain"
	"github.com/wyw14/cry-055/internal/repository/memory"
)

type adjustablePlanClock struct{ now time.Time }

func (c *adjustablePlanClock) Now() time.Time { return c.now }

func TestFutureScheduleRuleActivatesOnceOnEffectiveDate(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	clock := &adjustablePlanClock{now: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)}
	plan, err := domain.NewPlan("instrument-1", "item-1", time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC), "technician-1", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	plan.ScheduleRuleID = "rule-old"
	plan.ScheduleRuleEffectiveAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	rule := domain.ScheduleRule{
		ID: "rule-new", InstrumentID: plan.InstrumentID, ItemID: plan.ItemID,
		PeriodDays: 90, WarningDays: 14,
		EffectiveAt:  time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		SupersedesID: "rule-old",
	}
	lastQualifiedAt := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	service := NewPlanService(store, store, clock)

	unchanged, err := service.ApplyScheduleRule(ctx, plan.ID, rule, lastQualifiedAt, plan.Version)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Version != plan.Version || !unchanged.DueAt.Equal(plan.DueAt) || unchanged.ScheduleRuleID != "rule-old" {
		t.Fatalf("future rule changed plan early: due=%s rule=%s version=%d", unchanged.DueAt, unchanged.ScheduleRuleID, unchanged.Version)
	}

	clock.now = time.Date(2026, 12, 2, 9, 0, 0, 0, time.UTC)
	applied, err := service.ApplyScheduleRule(ctx, plan.ID, rule, lastQualifiedAt, plan.Version)
	if err != nil {
		t.Fatal(err)
	}
	wantDue := rule.EffectiveAt.AddDate(0, 0, rule.PeriodDays)
	if !applied.DueAt.Equal(wantDue) || applied.ScheduleRuleID != rule.ID || applied.Version != plan.Version.Next() {
		t.Fatalf("effective rule not applied correctly: due=%s rule=%s version=%d", applied.DueAt, applied.ScheduleRuleID, applied.Version)
	}

	repeated, err := service.ApplyScheduleRule(ctx, plan.ID, rule, lastQualifiedAt, applied.Version)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Version != applied.Version || !repeated.DueAt.Equal(applied.DueAt) {
		t.Fatalf("same rule was applied more than once: due=%s version=%d", repeated.DueAt, repeated.Version)
	}
}

// TestScheduleRuleDoesNotDriftOnRepeatedApplication reproduces the reported
// symptom: a periodic re-scan re-applies a rule that is already active. When
// lastQualifiedAt has moved past EffectiveAt (the realistic post-effective
// case, where the anchor is no longer clamped to EffectiveAt) NextDue would
// otherwise compute a different due date and the version would drift on every
// re-scan. The idempotency guard keyed on ScheduleRuleID keeps the plan stable.
func TestScheduleRuleDoesNotDriftOnRepeatedApplication(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	clock := &adjustablePlanClock{now: time.Date(2026, 12, 2, 9, 0, 0, 0, time.UTC)}
	plan, err := domain.NewPlan("instrument-1", "item-1", time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC), "technician-1", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreatePlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	rule := domain.ScheduleRule{
		ID: "rule-new", InstrumentID: plan.InstrumentID, ItemID: plan.ItemID,
		PeriodDays: 90, WarningDays: 14,
		EffectiveAt: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
	}
	// lastQualifiedAt is after EffectiveAt, so NextDue anchors on it directly;
	// shifting it later changes the computed due date unless re-application is
	// suppressed by the idempotency guard.
	lastQualifiedAt := time.Date(2027, 1, 5, 0, 0, 0, 0, time.UTC)
	service := NewPlanService(store, store, clock)

	applied, err := service.ApplyScheduleRule(ctx, plan.ID, rule, lastQualifiedAt, plan.Version)
	if err != nil {
		t.Fatal(err)
	}
	if applied.Version != plan.Version.Next() {
		t.Fatalf("rule not applied on first run: version=%d", applied.Version)
	}

	// A periodic re-scan happens after the qualified date shifts; without the
	// guard NextDue computes a different due date and the version drifts.
	shifted := lastQualifiedAt.AddDate(0, 0, 10)
	repeated, err := service.ApplyScheduleRule(ctx, plan.ID, rule, shifted, applied.Version)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Version != applied.Version || !repeated.DueAt.Equal(applied.DueAt) || repeated.ScheduleRuleID != applied.ScheduleRuleID {
		t.Fatalf("already-active rule drifted on re-application: due=%s version=%d rule=%s", repeated.DueAt, repeated.Version, repeated.ScheduleRuleID)
	}
}
