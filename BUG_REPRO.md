# Bug reproduction

## Bug

A future-dated schedule rule changes a manually rescheduled calibration plan before the rule becomes effective and increments the plan version prematurely.

## Trigger

Run:

```text
go test ./internal/application -run '^TestFutureScheduleRuleActivatesOnceOnEffectiveDate$' -count=1
```

## Observed error

```text
future rule changed plan early: due=2027-03-01 00:00:00 +0000 UTC rule=rule-new version=2
```
