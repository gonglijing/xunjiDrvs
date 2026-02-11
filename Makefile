# FSU Drivers Root Makefile
# Builds all driver WebAssembly modules under drvs/

.PHONY: all clean test install help list

# Compiler settings
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared
OPT ?= z

# Makefile location (works from any current directory)
ROOT_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

# Output directory (relative to project)
BUILD_DIR ?= $(ROOT_DIR)/../drivers

# Auto-discover all child driver Makefiles (two levels: site/device)
SUB_MAKEFILES := $(sort $(wildcard $(ROOT_DIR)/*/*/Makefile))
DRIVER_DIRS := $(patsubst %/,%,$(sort $(dir $(SUB_MAKEFILES))))

all:
	@if [ -z "$(DRIVER_DIRS)" ]; then \
		echo "No driver Makefiles found under $(ROOT_DIR)"; \
		exit 1; \
	fi
	@for dir in $(DRIVER_DIRS); do \
		echo "Building in $$dir"; \
		$(MAKE) -C "$$dir" all TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT) || exit 1; \
	done

install: all
	@mkdir -p "$(BUILD_DIR)"
	@find "$(ROOT_DIR)" -name "*.wasm" -type f ! -path "*/build/*" -exec cp {} "$(BUILD_DIR)/" \;
	@echo "Installed all wasm files to $(BUILD_DIR)/"

test:
	@for dir in $(DRIVER_DIRS); do \
		if [ -d "$$dir" ]; then \
			$(MAKE) -C "$$dir" test TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT) || true; \
		fi; \
	done

clean:
	@for dir in $(DRIVER_DIRS); do \
		$(MAKE) -C "$$dir" clean 2>/dev/null || true; \
	done
	@rm -rf "$(BUILD_DIR)"

list:
	@echo "Driver directories:"
	@for dir in $(DRIVER_DIRS); do echo " - $$dir"; done

help:
	@echo "FSU Drivers Build System"
	@echo ""
	@echo "Usage: make -f drvs/Makefile <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build all discovered drivers"
	@echo "  install - Build and copy wasm files to $(BUILD_DIR)"
	@echo "  test    - Run tests in all discovered drivers"
	@echo "  clean   - Clean all discovered drivers"
	@echo "  list    - List discovered driver directories"
	@echo "  help    - Show this message"
	@echo ""
	@echo "Discovered driver directories:"
	@for dir in $(DRIVER_DIRS); do echo "  $$dir"; done
