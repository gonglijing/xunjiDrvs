.PHONY: all clean test help

CARGO ?= cargo
TARGET ?= wasm32-unknown-unknown
PROFILE ?= release
CRATE ?= rust_modbus_rtu_template
OUTPUT ?= rust_modbus_rtu_template.wasm

MAKEFILE_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
TARGET_DIR := $(MAKEFILE_DIR)/target/$(TARGET)/$(PROFILE)

all: $(MAKEFILE_DIR)/$(OUTPUT)

$(MAKEFILE_DIR)/$(OUTPUT): $(MAKEFILE_DIR)/src/lib.rs $(MAKEFILE_DIR)/Cargo.toml
	cd $(MAKEFILE_DIR) && $(CARGO) build --target $(TARGET) --$(PROFILE)
	cp "$(TARGET_DIR)/$(CRATE).wasm" "$@"

test:
	cd $(MAKEFILE_DIR) && $(CARGO) test

clean:
	rm -rf "$(MAKEFILE_DIR)/target" "$(MAKEFILE_DIR)/$(OUTPUT)"

help:
	@echo "Rust Modbus RTU Driver Template"
	@echo ""
	@echo "Targets:"
	@echo "  all   - build $(OUTPUT)"
	@echo "  test  - run cargo test"
	@echo "  clean - remove build output"
