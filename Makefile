BINARY := grape
BIN_DIR := bin
CMD := ./cmd/gw

.PHONY: build clean

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) $(CMD)

clean:
	rm -f $(BIN_DIR)/$(BINARY)
