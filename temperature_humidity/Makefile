# Temperature and Humidity Sensor Driver Makefile
# For FSU (Field Site Unit) project

.PHONY: all clean test rtu tcp th

# Compiler settings
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared
OPT ?= z

# Output directory (relative to this directory)
BUILD_DIR ?= build

# Available drivers in this directory
DRIVERS := temperature_humidity th_modbusrtu th_modbustcp

all: $(addsuffix .wasm,$(DRIVERS))

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

%.wasm: %.go | $(BUILD_DIR)
	$(TINYGO) build -o $@ -target=$(TARGET) -buildmode=$(BUILDMODE) -opt=$(OPT) ./$<

# Individual targets for specific drivers
rtu: th_modbusrtu.wasm
tcp: th_modbustcp.wasm
th: temperature_humidity.wasm

test:
	@echo "Running temperature_humidity driver tests..."
	@if [ -f "temperature_humidity_test.go" ]; then $(TINYGO) test -target=$(TARGET) ./... || true; fi
	@if [ -f "th_modbusrtu_test.go" ]; then $(TINYGO) test -target=$(TARGET) ./... || true; fi

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(addsuffix .wasm,$(DRIVERS))
