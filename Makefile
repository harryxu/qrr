.PHONY: build release test check

build:
	go build -o bin/qrr ./cmd/qrr

release:
	go build -trimpath -ldflags='-s -w' -o bin/qrr ./cmd/qrr

test:
	go test ./...

check:
	go vet ./...
