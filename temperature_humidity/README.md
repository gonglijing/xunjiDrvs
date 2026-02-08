# Temperature and Humidity Sensors

This directory contains temperature and humidity sensor drivers for FSU.

## Available Sensors

- [Temperature & Humidity](README_temhumidity.md) - General temperature and humidity sensor
- [TH Modbus RTU](README_th_modbusrtu.md) - Modbus RTU temperature and humidity sensor
- [TH Modbus TCP](README_th_modbustcp.md) - Modbus TCP temperature and humidity sensor

## Supported Sensors

- Sagoo TH Sensor
- Industrial temperature and humidity transducers
- Modbus-connected environmental sensors

## Driver Structure

Each driver should include:
- `README.md` - Driver documentation
- `*.go` - Go implementation
- `*.wasm` - WebAssembly plugin (optional)

## Usage

Import and register the driver:
```go
import _ "fsu/drvs/temperature_humidity/your_driver"
```
