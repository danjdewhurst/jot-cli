.PHONY: build test lint clean install

BIN := bin/jot-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/danjdewhurst/jot-cli/cmd.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) .

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

INSTALL_DIR := $(HOME)/.local/bin

install: build
	mkdir -p $(INSTALL_DIR)
	rm -f $(INSTALL_DIR)/jot-cli
	cp $(BIN) $(INSTALL_DIR)/jot-cli
	ln -sf jot-cli $(INSTALL_DIR)/j
	@echo "Installed jot-cli and j to $(INSTALL_DIR)"
