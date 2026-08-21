# Bug reproduction

## Bug

Creating a calibration item against an invalid standard leaves the rejected item queryable, while equivalent instrument-model variants fail canonical applicability checks.

## Trigger

Run:

```text
go test ./internal/application -run '^TestCreateCalibrationItemValidatesApplicabilityAndRollsBackRejectedStandard$' -count=1
```

## Observed error

```text
rejected item must roll back, lookup error=<nil>
applicable models=[pg-10 PG-10 tp-20], want [PG-10 TP-20]
canonical applicability mismatch for [pg-10 PG-10 tp-20]
```
