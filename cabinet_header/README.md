# Cabinet Header Distribution Units

This directory contains cabinet header (power distribution unit) drivers for FSU.

## Available Devices

- Coming soon...

## Supported Types

- Intelligent PDU (Power Distribution Unit)
- Cabinet header monitors
- Branch circuit monitoring systems

## Driver Structure

Each driver should include:
- `README.md` - Driver documentation
- `*.go` - Go implementation
- `*.wasm` - WebAssembly plugin (optional)

## Usage

Import and register the driver:
```go
import _ "fsu/drvs/cabinet_header/your_driver"
```
