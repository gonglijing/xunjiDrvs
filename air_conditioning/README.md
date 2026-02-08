# Air Conditioning Drivers

This directory contains air conditioning (HVAC) device drivers for FSU.

## Available Drivers

- Coming soon...

## Driver Structure

Each driver should include:
- `README.md` - Driver documentation
- `*.go` - Go implementation
- `*.wasm` - WebAssembly plugin (optional)

## Usage

Import and register the driver:
```go
import _ "fsu/drvs/air_conditioning/your_driver"
```
