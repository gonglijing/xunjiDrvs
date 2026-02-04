.PHONY: all rtu tcp ups th install clean

ROOT := $(abspath ..)
DRIVERS_DIR ?= $(ROOT)/drivers
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared

DRIVER_SRCS := th_modbusrtu th_modbustcp ups_kstar temperature_humidity
WASMS := $(addsuffix .wasm,$(DRIVER_SRCS))

all: $(WASMS)

%.wasm: %.go
	$(TINYGO) build -o $@ -target=$(TARGET) -buildmode=$(BUILDMODE) ./$<

rtu: th_modbusrtu.wasm
tcp: th_modbustcp.wasm
ups: ups_kstar.wasm
th: temperature_humidity.wasm

install: all
	mkdir -p $(DRIVERS_DIR)
	cp $(WASMS) $(DRIVERS_DIR)/

clean:
	rm -f $(WASMS)
