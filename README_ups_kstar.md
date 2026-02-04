# 科士达 UPS Modbus TCP 驱动

## 设备信息

- **设备类型**: 科士达 UPS (不间断电源)
- **通信协议**: Modbus TCP
- **默认端口**: 502

## 点表配置

### 电池参数

| 字段名 | 功能码 | 寄存器地址 | 长度 | 数据类型 | 缩放系数 | 小数位数 | 读写 | 单位 | 说明 |
|--------|--------|------------|------|----------|----------|----------|------|------|------|
| battery_capacity | 0x03 | 100 | 1 | int16 | 0.1 | 1 | R | % | 电池容量 |
| battery_remain_time | 0x03 | 101 | 1 | int16 | 1 | 0 | R | min | 电池剩余时间 |

### 输入参数

| 字段名 | 功能码 | 寄存器地址 | 长度 | 数据类型 | 缩放系数 | 小数位数 | 读写 | 单位 | 说明 |
|--------|--------|------------|------|----------|----------|----------|------|------|------|
| input_frequency | 0x03 | 109 | 1 | int16 | 0.1 | 1 | R | Hz | 输入频率 |
| input_voltage_r | 0x03 | 110 | 1 | int16 | 0.1 | 1 | R | V | R相输入电压 |
| input_voltage_s | 0x03 | 111 | 1 | int16 | 0.1 | 1 | R | V | S相输入电压 |
| input_voltage_t | 0x03 | 112 | 1 | int16 | 0.1 | 1 | R | V | T相输入电压 |

### 输出参数

| 字段名 | 功能码 | 寄存器地址 | 长度 | 数据类型 | 缩放系数 | 小数位数 | 读写 | 单位 | 说明 |
|--------|--------|------------|------|----------|----------|----------|------|------|------|
| output_frequency | 0x03 | 119 | 1 | int16 | 0.1 | 1 | R | Hz | 输出频率 |
| output_voltage_r | 0x03 | 120 | 1 | int16 | 0.1 | 1 | R | V | R相输出电压 |
| output_voltage_s | 0x03 | 121 | 1 | int16 | 0.1 | 1 | R | V | S相输出电压 |
| output_voltage_t | 0x03 | 122 | 1 | int16 | 0.1 | 1 | R | V | T相输出电压 |
| load_percent_r | 0x03 | 123 | 1 | int16 | 1 | 0 | R | % | R相负载率 |
| load_percent_s | 0x03 | 124 | 1 | int16 | 1 | 0 | R | % | S相负载率 |
| load_percent_t | 0x03 | 125 | 1 | int16 | 1 | 0 | R | % | T相负载率 |

## 代码结构

```
ups_kstar.go
│
├── 【固定不变】Host 函数声明
│   └── tcp_transceive - TCP 发送接收接口
│
├── 【固定不变】配置结构
│   └── DriverConfig - 网关传入的配置
│
├── 【用户修改】点表定义
│   ├── REG_BATTERY_* - 电池参数寄存器地址
│   ├── REG_INPUT_* - 输入参数寄存器地址
│   ├── REG_OUTPUT_* - 输出参数寄存器地址
│   └── FUNC_CODE_READ - 功能码定义
│
├── 【用户修改】点表配置
│   └── pointConfig - 所有测点的详细配置
│
├── 【固定不变】驱动入口
│   ├── handle() - 读取数据入口
│   └── describe() - 描述可写字段
│
├── 【用户修改】读取所有测点
│   └── readAllUPS() - 按寄存器分组批量读取
│
├── 【固定不变】Modbus TCP 通信函数
│   ├── readSingleReg() - 读取单个寄存器
│   ├── readMultipleRegs() - 批量读取寄存器
│   ├── tcpTransceive() - TCP 发送接收
│   ├── buildReadRequest() - 构建读请求帧
│   └── parseReadResponse() - 解析读响应
│
└── 【固定不变】工具函数
    ├── getConfig() - 获取配置
    ├── formatFloat() - 格式化浮点数
    └── outputJSON() - 输出 JSON
```

## 性能优化

本驱动实现了批量读取优化：

- **输入参数**: 寄存器 109~112 连续，一次批量读取 (4个寄存器)
- **输出参数**: 寄存器 119~125 连续，一次批量读取 (7个寄存器)
- **电池参数**: 100, 101 非连续，分别单独读取

## 用户修改指南

### 1. 修改点表定义

```go
// 根据实际设备修改寄存器地址
const (
    REG_BATTERY_CAPACITY   = 100 // 电池容量
    REG_BATTERY_REMAIN_TIME = 101 // 电池剩余时间
    // ... 其他寄存器地址
)
```

### 2. 修改点表配置

```go
// 定义所有需要读取的测点
var pointConfig = []PointConfig{
    // 示例：添加新测点
    {Field: "new_point", Address: 200, Length: 1, Scale: 1.0, Decimals: 0, RW: "R", Unit: "unit", Label: "新测点"},
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

### 3. 修改批量读取逻辑

```go
// 在 readAllUPS() 函数中修改
// 根据寄存器地址分组实现批量读取优化

// 示例：添加新的批量读取组
if values := readMultipleRegs(byte(devAddr), 200, 5); values != nil {
    // 处理读取的5个连续寄存器
}
```

## 编译命令

```bash
# 编译为 WASM
tinygo build -o ups_kstar.wasm -target=wasip1 -buildmode=c-shared ./ups_kstar.go

# 安装到 drivers 目录
cp ups_kstar.wasm ../drivers/
```

## 网关配置

在网关管理界面中添加设备时，选择此驱动并配置：

- **设备地址**: Modbus 从站地址 (默认 1)
- **资源**: 网络资源 (IP:Port 格式，如 192.168.1.100:502)
