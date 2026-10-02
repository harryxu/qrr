.PHONY: build test check

build:
	go build -o bin/qrr ./cmd/qrr

test:
	go test ./...

check:
	go vet ./...
