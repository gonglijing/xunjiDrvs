# Air Conditioning Driver Makefile
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
DRIVERS :=

all: $(addsuffix .wasm,$(DRIVERS))

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

%.wasm: %.go | $(BUILD_DIR)
	$(TINYGO) build -o $@ -target=$(TARGET) -buildmode=$(BUILDMODE) -opt=$(OPT) ./$<

test:
	@echo "No tests defined for air_conditioning drivers yet"

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(addsuffix .wasm,$(DRIVERS))
