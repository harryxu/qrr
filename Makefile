.PHONY: build release test check

VERSION ?= dev

build:
	go build -ldflags="-X qrr/internal/cli.version=$(VERSION)" -o bin/qrr ./cmd/qrr

release:
	go build -trimpath -ldflags="-s -w -X qrr/internal/cli.version=$(VERSION)" -o bin/qrr ./cmd/qrr

test:
	go test ./...

check:
	go vet ./...
