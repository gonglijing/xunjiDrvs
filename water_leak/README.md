# Water Leak Detection Sensors

This directory contains water leak detection sensor drivers for FSU.

## Available Sensors

- Coming soon...

## Supported Types

- Spot leak detection sensors
- Cable-type water leak detection
- Pool-type leak sensors

## Driver Structure

Each driver should include:
- `README.md` - Driver documentation
- `*.go` - Go implementation
- `*.wasm` - WebAssembly plugin (optional)

## Usage

Import and register the driver:
```go
import _ "fsu/drvs/water_leak/your_driver"
```
