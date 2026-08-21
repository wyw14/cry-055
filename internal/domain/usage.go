package domain

import "time"

type UsageDecision string

const (
	UsageAllowed UsageDecision = "allowed"
	UsageBlocked UsageDecision = "blocked"
)

type UsageCheck struct {
	ID           ID               `json:"id"`
	InstrumentID ID               `json:"instrument_id"`
	BatchNumber  string           `json:"batch_number"`
	OperatorID   ID               `json:"operator_id"`
	Decision     UsageDecision    `json:"decision"`
	Status       InstrumentStatus `json:"status"`
	ReasonCode   string           `json:"reason_code,omitempty"`
	CheckedAt    time.Time        `json:"checked_at"`
}

func DecideUsage(instrument Instrument, batchNumber string, operatorID ID, now time.Time, warningDays int) UsageCheck {
	status := DerivedStatus(instrument.Status, now, instrument.NextDueAt, warningDays)
	check := UsageCheck{
		ID: NewID("use"), InstrumentID: instrument.ID, BatchNumber: batchNumber,
		OperatorID: operatorID, Decision: UsageAllowed, Status: status, CheckedAt: now.UTC(),
	}
	if !status.AllowsUse(instrument.Criticality == CriticalityCritical) {
		check.Decision = UsageBlocked
		check.ReasonCode = "INSTRUMENT_" + string(status)
	}
	return check
}
