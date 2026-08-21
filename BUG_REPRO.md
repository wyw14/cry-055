# Bug reproduction

## Bug

Instrument listing paginates before applying effective status and silently accepts invalid page input, producing incorrect totals and an unstable error contract.

## Trigger

Run:

```text
go test ./internal/transport/http -run '^TestInstrumentListFiltersEffectiveStatusBeforePaginationAndReturnsStableErrors$' -count=1
```

## Observed error

```text
unexpected page metadata: page=2 size=1 total=0
expected 422, got 200: {"items":null,"page":1,"size":1,"total":0}
```
