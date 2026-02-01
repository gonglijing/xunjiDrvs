# Modbus RTU 协议包

## 简介

`modbus` 包提供纯计算实现的 Modbus RTU 协议功能，可与 Extism WASM 驱动配合使用。

## 功能

- CRC-16/MODBUS 计算
- 请求帧构建
- 响应帧解析
- 数据类型转换

## API 文档

### CRC 计算

```go
// 计算 CRC-16/MODBUS 校验码
crc := modbus.CRC16(data []byte) uint16
```

### 请求帧构建

```go
// 构建 Modbus RTU 请求帧
frame := modbus.BuildRequestFrame(
    addr byte,      // 设备地址
    funcCode byte,  // 功能码
    regAddr uint16, // 寄存器地址
    count uint16,   // 寄存器数量
) []byte
```

### 响应帧解析

```go
// 解析读取响应
values, err := modbus.ParseReadResponse(
    response []byte, // 响应数据 (不含 CRC)
    addr byte,       // 期望的设备地址
    funcCode byte,   // 期望的功能码
) ([]uint16, error)

// 解析异常响应
exceptionCode, err := modbus.ParseReadResponseErr(
    response []byte,
    addr byte,
    funcCode byte,
) (byte, error)
```

### 数据类型转换

```go
// int16 值转换
value := modbus.Int16ToFloat64(raw uint16, scale float64) float64

// uint16 值转换
value := modbus.Uint16ToFloat64(raw uint16, scale float64) float64

// 合并两个 int16 为 int32
value := modbus.CombineInt16s(hi, lo uint16) int32

// 合并两个 uint16 为 uint32
value := modbus.CombineUint16s(hi, lo uint16) uint32
```

## 错误类型

```go
var (
    ErrInvalidResponse    = ModbusError("无效的响应数据")
    ErrAddrMismatch       = ModbusError("设备地址不匹配")
    ErrFuncCodeMismatch   = ModbusError("功能码不匹配")
    ErrCRCFail            = ModbusError("CRC 校验失败")
    ErrException          = ModbusError("设备返回异常")
)
```

## 在驱动中使用

```go
package main

import (
	"fmt"
	"unsafe"
	"./modbus"
)

//go:export serial_read
func serial_read(buf unsafe.Pointer, size int32) int32

//go:export serial_write
func serial_write(buf unsafe.Pointer, size int32) int32

//go:export collect
func collect() {
	// 构建请求帧
	req := modbus.BuildRequestFrame(1, 0x03, 0, 1)
	
	// 发送请求
	serial_write(unsafe.Pointer(&req[0]), int32(len(req)))
	
	// 解析响应
	values, err := modbus.ParseReadResponse(data, 1, 0x03)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	
	// 使用转换函数
	temp := modbus.Int16ToFloat64(values[0], 0.1)
	fmt.Printf("Temperature: %.1f\n", temp)
}
```
