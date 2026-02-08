# UPS Drivers

This directory contains Uninterruptible Power Supply (UPS) device drivers for FSU.

## Available Drivers

- [KStar UPS](README_ups_kstar.md) - KStar/NiBr series UPS devices

## Driver Structure

Each driver should include:
- `README.md` - Driver documentation
- `*.go` - Go implementation
- `*.wasm` - WebAssembly plugin (optional)

## Usage

Import and register the driver:
```go
import _ "fsu/drvs/ups/your_driver"
```
