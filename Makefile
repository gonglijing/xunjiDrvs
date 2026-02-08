# FSU Drivers Root Makefile
# Builds all device driver WebAssembly modules

.PHONY: all clean test help

# Compiler settings
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared
OPT ?= z

# Output directory (relative to parent fsu project)
BUILD_DIR ?= ../drivers

# Driver directories with their drivers
DRIVERS := \
	ups/ups_kstar \
	temperature_humidity/temperature_humidity \
	temperature_humidity/th_modbusrtu \
	temperature_humidity/th_modbustcp

# Default target - build all drivers
all:
	@for d in $(DRIVERS); do \
		dir=$${d%/*}; \
		name=$${d##*/}; \
		echo "Building $$name.wasm in $$dir/"; \
		$(MAKE) -C $$dir $$name.wasm TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT) || exit 1; \
	done

# Build specific driver categories
air_conditioning:
	@$(MAKE) -C air_conditioning all TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT)

ups:
	@$(MAKE) -C ups all TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT)

electric_meter:
	@$(MAKE) -C electric_meter all TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT)

temperature_humidity:
	@$(MAKE) -C temperature_humidity all TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT)

water_leak:
	@$(MAKE) -C water_leak all TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT)

cabinet_header:
	@$(MAKE) -C cabinet_header all TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT)

# Install all wasm files to parent fsu/drivers directory
install: all
	@mkdir -p $(BUILD_DIR)
	@find . -name "*.wasm" -type f ! -path "./build/*" -exec cp {} $(BUILD_DIR)/ \;
	@echo "Installed all wasm files to $(BUILD_DIR)/"

# Run tests for all drivers
test:
	@for dir in air_conditioning ups electric_meter temperature_humidity water_leak cabinet_header; do \
		if [ -d "$$dir" ]; then \
			$(MAKE) -C $$dir test || true; \
		fi; \
	done

# Clean all build artifacts
clean:
	@for dir in air_conditioning ups electric_meter temperature_humidity water_leak cabinet_header; do \
		$(MAKE) -C $$dir clean 2>/dev/null || true; \
	done
	rm -rf $(BUILD_DIR)
	rm -f *.wasm

# Show help
help:
	@echo "FSU Drivers Build System"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all                - Build all drivers (default)"
	@echo "  air_conditioning   - Build air conditioning drivers"
	@echo "  ups                - Build UPS drivers"
	@echo "  electric_meter     - Build electric meter drivers"
	@echo "  temperature_humidity - Build temperature/humidity drivers"
	@echo "  water_leak         - Build water leak detection drivers"
	@echo "  cabinet_header     - Build cabinet header/PDU drivers"
	@echo "  install            - Install all wasm files to parent drivers dir"
	@echo "  test               - Run tests for all drivers"
	@echo "  clean              - Remove all build artifacts"
	@echo "  help               - Show this help message"
	@echo ""
	@echo "Options:"
	@echo "  TINYGO=$(TINYGO)        - TinyGo compiler path"
	@echo "  TARGET=$(TARGET)        - WebAssembly target ($(TARGET))"
	@echo "  BUILDMODE=$(BUILDMODE) - Build mode ($(BUILDMODE))"
	@echo "  OPT=$(OPT)              - Optimization level ($(OPT))"
	@echo ""
	@echo "Drivers to build:"
	@echo "  $(DRIVERS)"
