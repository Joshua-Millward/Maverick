.PHONY: all plugin clean

GO := go
GOFLAGS := -buildmode=plugin -ldflags="-s -w"
GOEXPERIMENT := jsonv2,greenteagc
DIST_DIR := dist
SRC_SERVER := .

PLUGIN_FILES := \
	$(SRC_SERVER)/pl_lokycrypt.go \
	$(SRC_SERVER)/pl_main.go \
	$(SRC_SERVER)/pl_agent.go \
	$(SRC_SERVER)/pl_compile.go \
	$(SRC_SERVER)/pl_commands.go \
	$(SRC_SERVER)/pl_packer.go \
	$(SRC_SERVER)/pl_utils.go

PLUGIN_OUT := $(DIST_DIR)/maverick.so

all: clean plugin

plugin: $(PLUGIN_OUT)

$(PLUGIN_OUT): $(PLUGIN_FILES)
	@echo "[*] Building maverick plugin..."
	@mkdir -p $(DIST_DIR)
	@GOEXPERIMENT=$(GOEXPERIMENT) $(GO) build $(GOFLAGS) \
		-o $@ \
		$^
	@echo "[✓] Plugin built: $(PLUGIN_OUT)"

clean:
	@echo "[*] Cleaning build artifacts..."
	@if [ -f "$(PLUGIN_OUT)" ]; then \
		rm -f "$(PLUGIN_OUT)"; \
	fi
	@echo "[✓] Clean complete"
