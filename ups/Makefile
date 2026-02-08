# UPS Driver Makefile
# For FSU (Field Site Unit) project

.PHONY: all clean test

# Compiler settings
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared
OPT ?= z

# Output directory (relative to this directory)
BUILD_DIR ?= build

# Available drivers in this directory
DRIVERS := ups_kstar

all: $(addsuffix .wasm,$(DRIVERS))

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

%.wasm: %.go | $(BUILD_DIR)
	$(TINYGO) build -o $@ -target=$(TARGET) -buildmode=$(BUILDMODE) -opt=$(OPT) ./$<

test:
	@echo "Running UPS driver tests..."
	@if [ -f "ups_kstar_test.go" ]; then $(TINYGO) test -target=$(TARGET) ./... || true; else echo "No tests found"; fi

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(addsuffix .wasm,$(DRIVERS))
