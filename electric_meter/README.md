# Electric Meter Drivers

This directory contains electric meter device drivers for FSU.

## Available Drivers

- Coming soon...

## Supported Protocols

- Modbus RTU
- Modbus TCP
- DL/T 645
- IEC 62056

## Driver Structure

Each driver should include:
- `README.md` - Driver documentation
- `*.go` - Go implementation
- `*.wasm` - WebAssembly plugin (optional)

## Usage

Import and register the driver:
```go
import _ "fsu/drvs/electric_meter/your_driver"
```
