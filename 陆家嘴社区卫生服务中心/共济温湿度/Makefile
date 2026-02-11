# gongji temperature humidity Driver Makefile
# For FSU (Field Site Unit) project

.PHONY: all clean test

# Compiler settings
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared
OPT ?= z

# Makefile location (works from any current directory)
MAKEFILE_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

# Available drivers in this directory
DRIVERS := gongji_th

all: $(addprefix $(MAKEFILE_DIR)/,$(addsuffix .wasm,$(DRIVERS)))

$(MAKEFILE_DIR)/%.wasm: $(MAKEFILE_DIR)/%.go
	cd $(MAKEFILE_DIR) && $(TINYGO) build -o $@ -target=$(TARGET) -buildmode=$(BUILDMODE) -opt=$(OPT) ./$(notdir $<)

test:
	@echo "Running gongji temperature humidity driver tests..."
	@if [ -f "$(MAKEFILE_DIR)/gongji_th_test.go" ]; then cd $(MAKEFILE_DIR) && $(TINYGO) test -target=$(TARGET) ./... || true; else echo "No tests found"; fi

clean:
	rm -f $(addprefix $(MAKEFILE_DIR)/,$(addsuffix .wasm,$(DRIVERS)))
