# Bug reproduction

## Bug

Archiving multiple expired certificate attachments overwrites earlier bytes and keeps more than one attachment reader open at a time.

## Trigger

Run:

```text
go test ./internal/application -run '^TestArchiveExpiredCertificatesPreservesAttachmentsAndClosesReaders$' -count=1
```

## Observed error

```text
attachment readers were not released one at a time: peak=2 active=0
certificate archive mixed attachment contents: second-certificate-payloa | second-certificate-payload-is-longer
```
