# Temperature and Humidity Sensor Driver Makefile

.PHONY: all clean install rtu tcp th

ROOT := $(abspath ../..)
DRIVERS_DIR ?= $(ROOT)/drivers
TINYGO ?= tinygo
TARGET ?= wasip1
BUILDMODE ?= c-shared

# Available drivers in this directory
DRIVERS := temperature_humidity th_modbusrtu th_modbustcp

all: $(addsuffix .wasm,$(DRIVERS))

%.wasm: %.go
	$(TINYGO) build -o $@ -target=$(TARGET) -buildmode=$(BUILDMODE) -opt=z ./$<

rtu: th_modbusrtu.wasm
tcp: th_modbustcp.wasm
th: temperature_humidity.wasm

install: all
	mkdir -p $(DRIVERS_DIR)
	cp $(addsuffix .wasm,$(DRIVERS)) $(DRIVERS_DIR)/

clean:
	rm -f $(addsuffix .wasm,$(DRIVERS))
