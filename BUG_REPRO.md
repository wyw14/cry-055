# Bug reproduction

## Bug

Two concurrent alert scans for the same due instrument each create an alert, notification, and audit event instead of committing the business effect once.

## Trigger

Run:

```text
go test -race ./internal/application -run '^TestConcurrentAlertScanCommitsNotificationAndAuditOnce$' -count=1
```

## Observed error

```text
expected one scan to create the alert, got 2
expected one notification, got 2
expected one alert audit event, got 2
expected one open alert, total=2 items=2
```
