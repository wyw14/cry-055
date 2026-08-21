# Bug reproduction

## Bug

Concurrent retest and restoration actions can close a nonconformance before qualified retest evidence is committed.

## Trigger

Run:

```text
go test -race ./internal/application -run '^TestConcurrentRetestAndRestorationRequireQualifiedEvidence$' -count=1
```

## Observed error

```text
expected early restoration to be rejected, got <nil>
```
