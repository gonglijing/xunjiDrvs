# 温湿度传感器 Modbus RTU 驱动

## 硬件配置

| 参数 | 值 |
|------|-----|
| 串口 | /dev/cu.usbserial-130 |
| 波特率 | 9600 |
| 数据位 | 8 |
| 停止位 | 1 |
| 校验位 | None |
| 设备地址 | 1 |

## 寄存器定义 (硬编码在驱动内部)

| 寄存器地址 | 功能码 | 数据类型 | 有效值 | 测点名称 |
|-----------|--------|---------|--------|----------|
| 0 | 03 | int16 | v/10 | temperature |
| 1 | 03 | int16 | v/10 | humidity |

## 网关配置

驱动通过 `ConfigSchema` 字段获取配置，**仅需传递 resource_id 和 device_address**：

```json
{
    "resource_id": 1,
    "device_address": 1
}
```

### 配置参数说明

| 参数 | 类型 | 说明 |
|------|------|------|
| `resource_id` | int64 | 串口资源ID (关联网关资源配置) |
| `device_address` | int | Modbus 设备地址 (从机地址) |

## 输出格式

```json
{
  "success": true,
  "data": {
    "temperature": 25.3,
    "humidity": 60.5
  }
}
```

## 目录结构

```
drvs/
├── modbus/
│   ├── rtu.go        # Modbus RTU 协议包
│   └── README.md     # modbus 包文档
├── temperature_humidity.go  # 温湿度传感器驱动
└── README.md         # 本文档
```

## Modbus RTU 协议包

驱动使用独立的 `modbus` 包处理协议逻辑，包括：

- CRC-16/MODBUS 计算
- 请求帧构建 (`BuildRequestFrame`)
- 响应帧解析 (`ParseReadResponse`)
- 数据类型转换 (`Int16ToFloat64`, `Uint16ToFloat64`)

### 使用示例

```go
import (
	"./modbus"
)

// 构建请求帧
req := modbus.BuildRequestFrame(1, 0x03, 0, 1)

// 解析响应
values, err := modbus.ParseReadResponse(resp, 1, 0x03)

// 数据转换
temp := modbus.Int16ToFloat64(values[0], 0.1)
```

详见 [modbus/README.md](./modbus/README.md)

## 编译

需要安装 [TinyGo](https://tinygo.org/)：

```bash
# macOS 安装 TinyGo
brew install tinygo

# 编译为 WASM
cd drvs
tinygo build -o temperature_humidity.wasm -target=wasi -stack-size=64k temperature_humidity.go
```

## 适配其他设备

驱动中的寄存器定义在 `registers` 变量中，修改此变量即可适配不同设备：

```go
var registers = []registerDef{
    {addr: 0, name: "voltage", vtype: "uint16", scale: 0.1},  // 电压
    {addr: 1, name: "current", vtype: "uint16", scale: 0.01}, // 电流
}
```

同时需要修改 `outputJSON` 函数中的输出格式。

## 宿主集成

网关在加载驱动时自动注册以下 Host Functions：

| 函数名 | 参数 | 返回 | 说明 |
|--------|------|------|------|
| `serial_read` | buf, size | 实际读取字节数 | 从串口读取数据 |
| `serial_write` | buf, size | 实际写入字节数 | 向串口写入数据 |
| `sleep_ms` | ms | - | 毫秒级延时 |
| `output` | ptr, size | - | 输出字符串 |

## 部署

1. 编译生成 `.wasm` 文件
2. 通过网关管理界面上传驱动文件
3. 在创建设备时填写简化的 ConfigSchema 配置
