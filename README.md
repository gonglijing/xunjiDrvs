# HuShu 智能网关 - 驱动开发指南

## 项目概述

HuShu 智能网关是一个基于 Go 语言开发的工业物联网网关管理系统，采用 **Extism + TinyGo** 实现插件式驱动架构。

```
┌─────────────────────────────────────────────────────────────────────┐
│                     HuShu 智能网关系统架构                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐             │
│   │   前端 UI   │    │   HTTP API  │    │  采集调度   │             │
│   │  SolidJS    │◄──►│  Gorilla    │◄──►│  Collector  │             │
│   └─────────────┘    │    Mux      │    └──────┬──────┘             │
│                      └─────────────┘           │                     │
│                                                 ▼                     │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │                    DriverManager (驱动管理器)                │   │
│   │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │   │
│   │  │ WASM 插件   │◄─┤ Extism SDK  │◄─┤  Host Functions    │  │   │
│   │  │  Runtime    │  │             │  │ serial_transceive │  │   │
│   │  └─────────────┘  └─────────────┘  │  tcp_transceive    │  │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                          │                                          │
│                          ▼                                          │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │              WASM 驱动 (TinyGo 编写)                         │   │
│   │  th_modbusrtu.wasm  │  th_modbustcp.wasm  │  自定义驱动     │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

## 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| 网关运行时 | Go 1.21+ | 主程序语言 |
| 插件运行时 | Extism | WASM 插件框架 (go-sdk) |
| 驱动开发 | TinyGo | 编译为 WASM 字节码 |
| 协议栈 | Modbus RTU/TCP | 内置协议包 |
| 数据库 | SQLite | 配置 + 历史数据 |

## 目录结构

```
drvs/
├── README.md                    # 本文档
├── go.mod                       # 驱动模块依赖
├── temperature_humidity.wasm    # 示例驱动 (温湿度传感器)
├── th_modbusrtu.go              # Modbus RTU 驱动示例
├── th_modbustcp.go              # Modbus TCP 驱动示例
├── modbus/
│   ├── README.md               # Modbus 协议包文档
│   └── rtu.go                  # Modbus RTU 纯计算实现
└── ...
```

## 快速开始

### 环境要求

- **Go 1.21+** (用于编译网关)
- **TinyGo 0.40+** (用于编译 WASM 驱动)

### 安装 TinyGo

```bash
# macOS
brew install tinygo

# Linux (Debian/Ubuntu)
wget https://github.com/tinygo-org/tinygo/releases/download/v0.40.1/tinygo0.40.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf tinygo0.40.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/tinygo/bin

# 验证安装
tinygo version
```

### 编译驱动

```bash
cd drvs

# 编译 Modbus RTU 驱动
tinygo build -o th_modbusrtu.wasm -target=wasip1 -buildmode=c-shared th_modbusrtu.go

# 编译 Modbus TCP 驱动
tinygo build -o th_modbustcp.wasm -target=wasip1 -buildmode=c-shared th_modbustcp.go
```

## Host Functions 接口

网关为 WASM 驱动提供以下 Host Functions：

### TCP 通信

| 函数 | 参数 | 返回 | 说明 |
|------|------|------|------|
| `tcp_transceive` | wPtr: uint64, wSize: uint64, rPtr: uint64, rCap: uint64, timeoutMs: uint64 | uint64 | **写后读** |

### 输出与日志

驱动输出使用 Go PDK 的 `pdk.Output(...)` / `pdk.OutputJSON(...)`，
日志使用 `pdk.Log(...)`，无需自定义 `output` Host Function。

### serial_transceive 详解

**推荐使用** `serial_transceive` 替代单独的 read/write，它实现了完整的写后读流程：

```go
//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

// 使用示例
req := buildRequestFrame(1, 0x03, 0, 2)
resp := make([]byte, 64)
reqMem := pdk.AllocateBytes(req)
defer reqMem.Free()
respMem := pdk.Allocate(len(resp))
defer respMem.Free()
n := int(serial_transceive(
    reqMem.Offset(), uint64(len(req)),
    respMem.Offset(), uint64(len(resp)),
    300, // 300ms 超时
))
if n > 0 {
    mem := pdk.NewMemory(respMem.Offset(), uint64(n))
    mem.Load(resp[:n])
    // 处理响应数据
}
```

## 驱动开发

### 最小驱动模板

```go
package main

import (
	"encoding/json"

	pdk "github.com/extism/go-pdk"
)

// Host Functions
//go:wasmimport extism:host/user tcp_transceive
func tcp_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

// 驱动入口函数
//go:wasmexport handle
func handle() int32 {
    // 读取配置
    cfg := getConfig()
    
    // 读取设备数据
    points := readDevice(cfg)
    
    // 输出 JSON 结果
    outputJSON(map[string]interface{}{
        "success": true,
        "points":  points,
    })
    return 0
}

// 可选：描述驱动的可写字段
//go:wasmexport describe
func describe() int32 {
    outputJSON(map[string]interface{}{
        "success": true,
        "data": map[string]interface{}{
            "writable": []map[string]interface{}{
                {"field": "field_name", "label": "标签", "desc": "描述"},
            },
        },
    })
    return 0
}

// 驱动版本（网关会读取此版本用于展示）
const DriverVersion = "1.0.0"

//go:wasmexport version
func version() int32 {
    outputJSON(map[string]interface{}{
        "success": true,
        "data": map[string]string{
            "version": DriverVersion,
        },
    })
    return 0
}

// 从网关配置中获取参数
func getConfig() GatewayConfig {
    // 使用 PDK 读取输入 JSON
    var envelope struct { Config GatewayConfig `json:"config"` }
    _ = pdk.InputJSON(&envelope)
    return GatewayConfig{DeviceAddress: 1, FuncName: "read"}
}

// 读取设备数据
func readDevice(cfg GatewayConfig) []map[string]interface{} {
    // 实现设备通信逻辑
    return []map[string]interface{}{
        {"field_name": "temperature", "value": 25.3, "rw": "R"},
    }
}

// 输出 JSON 字符串
func outputJSON(v interface{}) {
    b, _ := json.Marshal(v)
    pdk.Output(b)
}

func main() {}
```

### 配置传递

驱动通过 `Config` 结构接收网关配置：

```go
type GatewayConfig struct {
    DeviceAddress int    `json:"device_address"` // 设备地址
    FuncName      string `json:"func_name"`      // "read" | "write"
    FieldName     string `json:"field_name"`     // 写操作时指定字段
    Value         float64 `json:"value"`         // 写操作时的值
}
```

网关在调用驱动时自动传递设备配置，驱动通过 `pdk.InputJSON` 获取。

### 输出格式

**推荐格式** (支持多点读写)：

```json
{
  "success": true,
  "points": [
    {"field_name": "temperature", "value": 25.3, "rw": "R"},
    {"field_name": "humidity", "value": 60.5, "rw": "R"}
  ]
}
```

**写操作响应**：

```json
{
  "success": true,
  "data": {
    "field": "temperature",
    "value": "30.0"
  }
}
```

**错误响应**：

```json
{
  "success": false,
  "error": "设备无响应"
}
```

## Modbus RTU 驱动示例

`th_modbusrtu.go` 是一个完整的温湿度传感器驱动示例：

```go
// 寄存器定义
var registers = []registerDef{
    {addr: 0x0000, name: "temperature", scale: 0.1, rw: "R"}, // 温度
    {addr: 0x0001, name: "humidity", scale: 0.1, rw: "R"},    // 湿度
    {addr: 0x0021, name: "device_addr", rw: "W"},             // 设备地址 (可写)
    {addr: 0x0022, name: "temp_alarm_threshold", rw: "RW"},   // 温度阈值
}

// 读取所有寄存器
func readAll(devAddr int) ([]map[string]interface{}, bool) {
    points := make([]map[string]interface{}, 0)
    
    // 读取温湿度 (0x0000~0x0001)
    req := buildReadFrame(byte(devAddr), 0x0000, 0x0002)
    resp := make([]byte, 32)
    reqMem := pdk.AllocateBytes(req)
    defer reqMem.Free()
    respMem := pdk.Allocate(len(resp))
    defer respMem.Free()
n := int(serial_transceive(
    reqMem.Offset(), uint64(len(req)),
    respMem.Offset(), uint64(len(resp)),
    300,
))

mem := pdk.NewMemory(respMem.Offset(), uint64(n))
mem.Load(resp[:n])
if ps, err := decodeMulti(resp[:n]); err == nil {
    points = append(points, ps...)
}
    
    return points, len(points) > 0
}

// 写寄存器
func doWrite(cfg GatewayConfig) bool {
    switch cfg.FieldName {
    case "device_addr":
        return writeRegister(cfg.DeviceAddress, 0x0021, uint16(cfg.Value))
    case "temp_alarm_threshold":
        return writeRegister(cfg.DeviceAddress, 0x0022, uint16(cfg.Value))
    }
    return false
}
```

## Modbus TCP 驱动示例

`th_modbustcp.go` 使用 `tcp_transceive` 实现 Modbus TCP 通信：

```go
//go:wasmimport extism:host/user tcp_transceive
func tcp_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

func readHoldingRegisters(cfg GatewayConfig) []map[string]interface{} {
    // 构建 Modbus TCP 请求
    req := buildMBAPHeader(cfg.DeviceAddress, 0x03, 0x0000, 0x0002)
    
    // 发送并接收
    resp := make([]byte, 64)
    reqMem := pdk.AllocateBytes(req)
    defer reqMem.Free()
    respMem := pdk.Allocate(len(resp))
    defer respMem.Free()
n := int(tcp_transceive(
    reqMem.Offset(), uint64(len(req)),
    respMem.Offset(), uint64(len(resp)),
    500,
))

// 解析响应
mem := pdk.NewMemory(respMem.Offset(), uint64(n))
mem.Load(resp[:n])
return parseResponse(resp[:n])
}
```

## Modbus 协议包

`modbus` 包提供纯计算实现的 Modbus RTU 协议功能：

```go
import "./modbus"

// 计算 CRC
crc := modbus.CRC16(data)

// 构建请求帧
frame := modbus.BuildRequestFrame(addr, 0x03, start, count)

// 解析响应
values, err := modbus.ParseReadResponse(resp, addr, 0x03)

// 数据转换
temp := modbus.Int16ToFloat64(values[0], 0.1)
```

### API 参考

| 函数 | 说明 |
|------|------|
| `CRC16(data []byte) uint16` | 计算 CRC-16/MODBUS |
| `BuildRequestFrame()` | 构建读取请求帧 |
| `BuildWriteFrame()` | 构建写单个寄存器请求 |
| `ParseReadResponse()` | 解析读取响应 |
| `Int16ToFloat64()` | int16 缩放转换 |
| `Uint16ToFloat64()` | uint16 缩放转换 |

## 部署流程

```
1. 编写驱动源码 (.go)
           │
           ▼
2. TinyGo 编译为 WASM
   tinygo build -o xxx.wasm -target=wasip1 -buildmode=c-shared xxx.go
           │
           ▼
3. 网管管理界面上传
   /drivers → 上传驱动文件
           │
           ▼
4. 创建设备并关联驱动
   /devices → 选择驱动
           │
           ▼
5. 启用设备 → 自动采集数据
```

## 编译选项

| 选项 | 说明 | 推荐值 |
|------|------|--------|
| `-target=wasip1` | WASI Preview1 | 必需 |
| `-stack-size=64k` | 栈大小 | 64k~128k |
| `-opt=z` | 优化级别 | z (最小体积) |

完整编译命令：

```bash
tinygo build -o th_modbusrtu.wasm \
    -target=wasip1 \
    -buildmode=c-shared \
    -stack-size=64k \
    -opt=z \
    th_modbusrtu.go
```

## 故障排查

### 常见问题

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| 读取超时 | 串口占用/地址错误 | 检查设备地址、确认串口未被占用 |
| CRC 校验失败 | 波特率不匹配 | 确认通信参数一致 |
| 驱动加载失败 | WASM 格式错误 | 重新编译、检查堆栈大小 |
| 数据解析错误 | 寄存器定义错误 | 对照设备手册确认寄存器地址 |

### 调试技巧

```go
// 使用 output 函数输出调试信息
func debugPoint(name string, value float64) {
    outputJSON(map[string]interface{}{
        "debug": map[string]interface{}{
            "point":  name,
            "value": value,
        },
    })
}
```

## 性能优化

1. **使用 serial_transceive**: 避免单独 read/write 的竞态条件
2. **合理设置超时**: 根据设备响应时间调整 timeoutMs
3. **减少内存分配**: 复用 buffer 减少 GC 压力
4. **批量读取**: 一次请求读取多个寄存器

## 相关文档

- [网关主 README](../README.md)
- [Extism 文档](https://extism.org/)
- [TinyGo 文档](https://tinygo.org/)
- [Modbus 协议规范](https://modbus.org/)

## 许可证

MIT License
| `-buildmode=c-shared` | Go PDK 需要 | 必需 |
