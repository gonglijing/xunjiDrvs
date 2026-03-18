.PHONY: all clean test help

CARGO ?= cargo
TARGET ?= wasm32-unknown-unknown
PROFILE ?= release
CRATE ?= rust_extism_demo

MAKEFILE_DIR := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
TARGET_DIR := $(MAKEFILE_DIR)/target/$(TARGET)/$(PROFILE)
OUTPUT := $(MAKEFILE_DIR)/$(CRATE).wasm

all: $(OUTPUT)

$(OUTPUT): $(MAKEFILE_DIR)/src/lib.rs $(MAKEFILE_DIR)/Cargo.toml
	cd $(MAKEFILE_DIR) && $(CARGO) build --target $(TARGET) --$(PROFILE)
	cp "$(TARGET_DIR)/$(CRATE).wasm" "$(OUTPUT)"

test:
	cd $(MAKEFILE_DIR) && $(CARGO) test

clean:
	rm -rf "$(MAKEFILE_DIR)/target" "$(OUTPUT)"

help:
	@echo "Rust Extism Demo Driver"
	@echo ""
	@echo "Targets:"
	@echo "  all   - build $(CRATE).wasm"
	@echo "  test  - run cargo test"
	@echo "  clean - remove build output"
