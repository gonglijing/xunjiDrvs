# FSU Drivers Root Makefile
# Builds all driver WebAssembly modules under drvs/

.PHONY: all tinygo rust clean test test-tinygo test-rust install help list list-tinygo list-rust

# Compiler settings
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared
OPT ?= z

# Makefile location (works from any current directory)
ROOT_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

# Output directory (relative to project)
BUILD_DIR ?= $(ROOT_DIR)/../drivers

TINYGO_ROOT := $(ROOT_DIR)/tinygo
RUST_ROOT := $(ROOT_DIR)/rust

# Auto-discover driver Makefiles
TINYGO_MAKEFILES := $(sort $(wildcard $(TINYGO_ROOT)/*/*/Makefile))
TINYGO_DRIVER_DIRS := $(patsubst %/,%,$(sort $(dir $(TINYGO_MAKEFILES))))

RUST_MAKEFILES := $(sort $(wildcard $(RUST_ROOT)/*/*/Makefile))
RUST_DRIVER_DIRS := $(patsubst %/,%,$(sort $(dir $(RUST_MAKEFILES))))

all: tinygo rust

tinygo:
	@if [ -z "$(TINYGO_DRIVER_DIRS)" ]; then \
		echo "No TinyGo driver Makefiles found under $(TINYGO_ROOT)"; \
		exit 1; \
	fi
	@for dir in $(TINYGO_DRIVER_DIRS); do \
		echo "Building TinyGo driver in $$dir"; \
		$(MAKE) -C "$$dir" all TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT) || exit 1; \
	done

rust:
	@if [ -z "$(RUST_DRIVER_DIRS)" ]; then \
		echo "No Rust driver Makefiles found under $(RUST_ROOT)"; \
		exit 1; \
	fi
	@for dir in $(RUST_DRIVER_DIRS); do \
		echo "Building Rust driver in $$dir"; \
		$(MAKE) -C "$$dir" all || exit 1; \
	done

install: all
	@mkdir -p "$(BUILD_DIR)"
	@find "$(TINYGO_ROOT)" "$(RUST_ROOT)" -name "*.wasm" -type f ! -path "*/build/*" ! -path "*/target/*" -exec cp {} "$(BUILD_DIR)/" \;
	@echo "Installed all wasm files to $(BUILD_DIR)/"

test: test-tinygo test-rust

test-tinygo:
	@for dir in $(TINYGO_DRIVER_DIRS); do \
		if [ -d "$$dir" ]; then \
			$(MAKE) -C "$$dir" test TINYGO=$(TINYGO) TARGET=$(TARGET) BUILDMODE=$(BUILDMODE) OPT=$(OPT) || true; \
		fi; \
	done

test-rust:
	@for dir in $(RUST_DRIVER_DIRS); do \
		if [ -d "$$dir" ]; then \
			$(MAKE) -C "$$dir" test || true; \
		fi; \
	done

clean:
	@for dir in $(TINYGO_DRIVER_DIRS) $(RUST_DRIVER_DIRS); do \
		$(MAKE) -C "$$dir" clean 2>/dev/null || true; \
	done
	@rm -rf "$(BUILD_DIR)"

list:
	@$(MAKE) -f "$(ROOT_DIR)/Makefile" list-tinygo
	@$(MAKE) -f "$(ROOT_DIR)/Makefile" list-rust

list-tinygo:
	@echo "TinyGo driver directories:"
	@for dir in $(TINYGO_DRIVER_DIRS); do echo " - $$dir"; done

list-rust:
	@echo "Rust driver directories:"
	@for dir in $(RUST_DRIVER_DIRS); do echo " - $$dir"; done

help:
	@echo "FSU Drivers Build System"
	@echo ""
	@echo "Usage: make -f drvs/Makefile <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all         - Build all TinyGo and Rust drivers"
	@echo "  tinygo      - Build all TinyGo drivers"
	@echo "  rust        - Build all Rust drivers"
	@echo "  install     - Build and copy wasm files to $(BUILD_DIR)"
	@echo "  test        - Run tests for all drivers"
	@echo "  test-tinygo - Run tests for TinyGo drivers"
	@echo "  test-rust   - Run tests for Rust drivers"
	@echo "  clean       - Clean all discovered drivers"
	@echo "  list        - List all discovered driver directories"
	@echo "  help        - Show this message"
	@echo ""
	@echo "Discovered TinyGo driver directories:"
	@for dir in $(TINYGO_DRIVER_DIRS); do echo "  $$dir"; done
	@echo ""
	@echo "Discovered Rust driver directories:"
	@for dir in $(RUST_DRIVER_DIRS); do echo "  $$dir"; done
