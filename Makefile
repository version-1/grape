BINARY := grape
BIN_DIR := bin
CMD := ./cmd/grape
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
RELEASE_DIR ?= dist
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build release clean

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD)

release:
	VERSION="$(patsubst v%,%,$(VERSION))" COMMIT="$(COMMIT)" RELEASE_DIR="$(RELEASE_DIR)" sh scripts/release.sh

clean:
	rm -f $(BIN_DIR)/$(BINARY)
