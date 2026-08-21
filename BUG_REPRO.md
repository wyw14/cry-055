# Bug reproduction

## Bug

A failed calibration revision link leaves an orphan revision, breaks immutable history metadata, and loses the causal error identity.

## Trigger

Run:

```text
go test ./internal/application -run '^TestRevisionFailurePreservesImmutableVersionChain$' -count=1
```

## Observed error

```text
first revision broke chain metadata
expected link failure in error chain, got link calibration revision: revision head unavailable
failed revision survived rollback
expected three committed versions, got 2
```
