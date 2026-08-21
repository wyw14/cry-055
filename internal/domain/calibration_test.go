package domain

import (
	"reflect"
	"testing"
	"time"
)

func TestExecutionConclusionUsesEveryMeasurement(t *testing.T) {
	now := time.Now().UTC()
	measurements := []Measurement{{Point: "zero", Passed: true}, {Point: "span", Passed: false}}
	execution, err := NewExecution("ins", "item", "plan", "operator", measurements, "", now, now)
	if err != nil {
		t.Fatal(err)
	}
	if execution.Conclusion != ConclusionUnqualified {
		t.Fatalf("expected unqualified, got %s", execution.Conclusion)
	}
}
func TestRevisionCreatesNewVersionIdentity(t *testing.T) {
	now := time.Now().UTC()
	original, err := NewExecution("ins", "item", "plan", "operator", []Measurement{{Point: "zero", Passed: true}}, "", now, now)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := original.Revise([]Measurement{{Point: "zero", Passed: true}}, "reviewer", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if revision.ID == original.ID || revision.PreviousID != original.ID {
		t.Fatal("revision did not preserve immutable version chain")
	}
}
func TestReviewRequiresIndependentActor(t *testing.T) {
	now := time.Now().UTC()
	execution, _ := NewExecution("ins", "item", "plan", "same", []Measurement{{Point: "zero", Passed: true}}, "", now, now)
	if err := execution.Review("same", "", now); err == nil {
		t.Fatal("executor reviewed their own result")
	}
}

func TestNewCalibrationItemCanonicalizesApplicableModels(t *testing.T) {
	now := time.Now().UTC()
	item, err := NewCalibrationItem("TEMP", "Temperature verification", 90, 10, 0.2, "C", "std", []string{" pg-10 ", "PG-10", "tp-20", "  ", ""}, now)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"PG-10", "TP-20"}
	if !reflect.DeepEqual(item.ApplicableModels, want) {
		t.Fatalf("applicable models=%v, want %v", item.ApplicableModels, want)
	}
}

func TestAppliesToMatchesCaseAndSpaceVariants(t *testing.T) {
	now := time.Now().UTC()
	item, _ := NewCalibrationItem("TEMP", "Temperature verification", 90, 10, 0.2, "C", "std", []string{"PG-10", "TP-20"}, now)
	cases := map[string]bool{
		"pg-10":      true,
		"  PG-10 ":   true,
		"tp-20":      true,
		"humidity-9": false,
		"":           false,
	}
	for model, expected := range cases {
		if got := item.AppliesTo(model); got != expected {
			t.Errorf("AppliesTo(%q)=%v, want %v", model, got, expected)
		}
	}
	empty, _ := NewCalibrationItem("ANY", "Any item", 90, 10, 0.2, "C", "std", nil, now)
	if !empty.AppliesTo("anything") {
		t.Error("empty applicable models list should match any model")
	}
}
