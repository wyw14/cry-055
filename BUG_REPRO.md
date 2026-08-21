# Bug reproduction

## Bug

When the audit append for an approved execution review fails, the review decision remains persisted and callers can no longer match the original audit failure.

## Trigger

Run:

```text
go test ./internal/application -run '^TestReviewAuditFailurePreservesCauseAndRollsBackDecision$' -count=1
```

## Observed error

```text
review error lost audit cause: review execution failed during audit append: audit sink unavailable
failed review left persisted decision: reviewer=reviewer-1 version=2
```
