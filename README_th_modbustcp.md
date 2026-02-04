# 温湿度传感器 Modbus TCP 驱动

## 设备信息

- **设备类型**: 温湿度传感器
- **通信协议**: Modbus TCP
- **默认端口**: 502

## 点表配置

| 字段名 | 功能码 | 寄存器地址 | 长度 | 数据类型 | 缩放系数 | 小数位数 | 读写 | 单位 | 说明 |
|--------|--------|------------|------|----------|----------|----------|------|------|------|
| temperature | 0x03 | 0x0000 | 1 | int16 | 0.1 | 1 | R | °C | 温度 |
| humidity | 0x03 | 0x0001 | 1 | int16 | 0.1 | 1 | R | % | 湿度 |
| temp_alarm_threshold | 0x03 | 0x0022 | 1 | int16 | 1 | 0 | RW | °C | 温度告警阈值 |

## 代码结构

```
th_modbustcp.go
│
├── 【固定不变】Host 函数声明
│   └── tcp_transceive - TCP 发送接收接口
│
├── 【固定不变】配置结构
│   └── DriverConfig - 网关传入的配置
│
├── 【用户修改】点表定义
│   ├── REG_TEMPERATURE - 温度寄存器地址
│   ├── REG_HUMIDITY - 湿度寄存器地址
│   ├── REG_TEMP_ALARM_THRESHOLD - 温度告警阈值寄存器地址
│   └── FUNC_CODE_READ/WRITE - 功能码定义
│
├── 【用户修改】点表配置
│   └── pointConfig - 所有测点的详细配置
│
├── 【固定不变】驱动入口
│   ├── handle() - 读取/写入数据入口
│   └── describe() - 描述可写字段
│
├── 【用户修改】读取所有测点
│   └── readAllPoints() - 根据点表配置批量读取
│
├── 【用户修改】写操作
│   └── doWrite() - 写入寄存器逻辑
│
├── 【固定不变】Modbus TCP 通信函数
│   ├── tcpTransceive() - TCP 发送接收
│   ├── buildReadRequest() - 构建读请求帧
│   ├── buildWriteRequest() - 构建写请求帧
│   ├── parseReadResponse() - 解析读响应
│   └── writeRegister() - 写单个寄存器
│
└── 【固定不变】工具函数
    ├── getConfig() - 获取配置
    ├── formatFloat() - 格式化浮点数
    └── outputJSON() - 输出 JSON
```

## 用户修改指南

### 1. 修改点表定义

```go
// 根据实际设备修改寄存器地址
const (
    REG_TEMPERATURE         = 0x0000 // 温度寄存器地址
    REG_HUMIDITY            = 0x0001 // 湿度寄存器地址
    REG_TEMP_ALARM_THRESHOLD = 0x0022 // 温度告警阈值寄存器地址
)
```

### 2. 修改点表配置

```go
// 定义所有需要读取的测点
var pointConfig = []PointConfig{
    // 示例：添加新测点
    {Field: "new_point", Address: 0x0002, Length: 1, Scale: 1.0, Decimals: 0, RW: "R", Unit: "unit", Label: "新测点"},
}
```

### PointConfig 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| Field | string | 字段名，用于数据库存储 |
| Address | uint16 | 寄存器地址 (Modbus 地址) |
| Length | uint16 | 寄存器数量 |
| Scale | float64 | 缩放系数 (原始值 × Scale = 实际值) |
| Decimals | int | 有效小数位数 |
| RW | string | 读写属性 ("R" 或 "RW") |
| Unit | string | 单位 |
| Label | string | 显示标签 |

### 3. 修改写操作逻辑

```go
// 在 doWrite() 函数中根据实际设备修改
func doWrite(cfg DriverConfig) bool {
    switch cfg.FieldName {
    case "field_name1":
        // 写字段1的逻辑
        return writeRegister(...)
    case "field_name2":
        // 写字段2的逻辑
        return writeRegister(...)
    default:
        return false
    }
}
```

## 编译命令

```bash
# 编译为 WASM
tinygo build -o th_modbustcp.wasm -target=wasip1 -buildmode=c-shared ./th_modbustcp.go

# 安装到 drivers 目录
cp th_modbustcp.wasm ../drivers/
```

## 网关配置

在网关管理界面中添加设备时，选择此驱动并配置：

- **设备地址**: Modbus 从站地址 (默认 1)
- **资源**: 网络资源 (IP:Port 格式，如 192.168.1.100:502)
