.PHONY: start build test install

BIN := bin/kairos

# Run the CLI via go run (optional: make start ARGS="status")
start:
	go run ./cmd/kairos $(ARGS)

build:
	mkdir -p $(dir $(BIN))
	go build -o $(BIN) ./cmd/kairos

test:
	go test ./...

# Install kairos into Go bin ($GOBIN or $GOPATH/bin); see install.sh
install:
	sh ./install.sh
