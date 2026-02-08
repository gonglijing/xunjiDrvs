# Electric Meter Driver Makefile

.PHONY: all clean install

ROOT := $(abspath ../..)
DRIVERS_DIR ?= $(ROOT)/drivers
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared

# Available drivers in this directory
DRIVERS :=

all: $(addsuffix .wasm,$(DRIVERS))

%.wasm: %.go
	$(TINYGO) build -o $@ -target=$(TARGET) -buildmode=$(BUILDMODE) -opt=z ./$<

install: all
	mkdir -p $(DRIVERS_DIR)
	cp $(addsuffix .wasm,$(DRIVERS)) $(DRIVERS_DIR)/

clean:
	rm -f $(addsuffix .wasm,$(DRIVERS))
