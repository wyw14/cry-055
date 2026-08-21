# Bug reproduction

## Bug

Canceling an instrument usage check before repository access finishes still returns a blocked decision and persists usage and audit side effects.

## Trigger

Run:

```text
go test ./internal/transport/http -run '^TestUsageCheckHTTPCancellationLeavesNoDecisionOrAudit$' -count=1
```

## Observed error

```text
expected canceled request status 499, got 423
unexpected cancellation code: INSTRUMENT_BLOCKED
canceled check leaked side effects: checks=1 audits=1
```
