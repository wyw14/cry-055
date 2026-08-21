SHELL := pwsh.exe -NoProfile -Command

.PHONY: build test race vet web-build run docker-build

build:
	go build ./...

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

web-build:
	Set-Location web; npm ci; npm run build

run:
	go run ./cmd/server

docker-build:
	docker build -t cry-055-backend .
