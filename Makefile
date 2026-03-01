.PHONY: build test lint clean install

BIN := bin/jot
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/duncanjbrown/jot-cli/cmd.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) .

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

install: build
	cp $(BIN) $(GOPATH)/bin/jot 2>/dev/null || cp $(BIN) ~/go/bin/jot
