# Bug reproduction

## Bug

A fully assessed nonconformance with a qualified retest cannot be restored when the instrument has no next due date, leaving it stuck in reinspection.

## Trigger

Run:

```text
go test ./internal/application -run '^TestRestoreAuthorizesQualityManagerWithPartialInstrumentState$' -count=1
```

## Observed error

```text
restore fully reviewed case: invalid state transition: effective status is unavailable
```
