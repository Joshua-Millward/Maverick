.PHONY: all clean

GO := go
GOFLAGS := -buildmode=plugin -ldflags="-s -w"
GOEXPERIMENT := jsonv2,greenteagc
DIST_DIR := dist
OUTPUT_FILE := $(DIST_DIR)/maverick_listener.so
SOURCE_FILES := \
	pl_lokycrypt.go \
	pl_main.go \
	pl_http.go \
	pl_utils.go \
	pl_structs.go

all: clean
	@ echo "    * Building maverick_listener plugin"
	@ mkdir -p $(DIST_DIR)
	@ GOEXPERIMENT=$(GOEXPERIMENT) $(GO) build $(GOFLAGS) -o $(OUTPUT_FILE) $(SOURCE_FILES)
	@ echo "      done..."

clean:
	@ echo "    * Cleaning build artifacts..."
	@if [ -f "$(OUTPUT_FILE)" ]; then \
		rm -f "$(OUTPUT_FILE)"; \
	fi
	@ echo "      done..."
