.PHONY: start build test install

BIN := bin/kairos
VERSION := $(shell git describe --tags --always --dirty)

# Run the CLI via go run (optional: make start ARGS="status")
start:
	go run . $(ARGS)

build:
	mkdir -p $(dir $(BIN))
	go build -ldflags "-X github.com/shadmeoli/kairos/internal/version.Tag=$(VERSION)" -o $(BIN) .

test:
	go test ./internal/gitexec -coverprofile=coverage.out
	go tool cover -func=coverage.out
	rm -f coverage.out

# Install kairos into Go bin ($GOBIN or $GOPATH/bin); see install.sh
install:
	sh ./install.sh
