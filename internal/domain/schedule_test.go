package domain

import (
	"testing"
	"time"
)

func TestScheduleRuleChangeUsesEffectiveDate(t *testing.T) {
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	effective := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	rule := ScheduleRule{InstrumentID: "ins", ItemID: "item", PeriodDays: 90, WarningDays: 15, EffectiveAt: effective}
	due, err := rule.NextDue(old)
	if err != nil {
		t.Fatal(err)
	}
	want := effective.AddDate(0, 0, 90)
	if !due.Equal(want) {
		t.Fatalf("want %s got %s", want, due)
	}
}
