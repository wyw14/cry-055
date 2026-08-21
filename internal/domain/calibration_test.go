package domain

import (
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
