# FSU Device Drivers

FSU（场站单元）设备驱动目录，基于 **Extism + TinyGo** 的 WebAssembly 插件架构。

## 目录结构

```
drvs/
├── Makefile              # 统一构建入口
├── go.mod                # 驱动模块依赖
├── air_conditioning/     # 空调驱动（暂无）
├── ups/                  # UPS 驱动
│   └── ups_kstar/       # 科士达 UPS
├── electric_meter/       # 电表驱动（暂无）
├── temperature_humidity/ # 温湿度传感器驱动
│   ├── temperature_humidity/  # 温湿度传感器
│   ├── th_modbusrtu/    # Modbus RTU 版
│   └── th_modbustcp/    # Modbus TCP 版
├── water_leak/          # 水浸传感器驱动（暂无）
└── cabinet_header/      # 机柜 PDU 驱动（暂无）
```

## 驱动状态

| 目录 | 驱动名称 | 协议 | 状态 |
|------|---------|------|------|
| ups | ups_kstar | Modbus TCP | ✅ 已实现 |
| temperature_humidity | temperature_humidity | Modbus RTU | ✅ 已实现 |
| temperature_humidity | th_modbusrtu | Modbus RTU | ✅ 已实现 |
| temperature_humidity | th_modbustcp | Modbus TCP | ✅ 已实现 |
| air_conditioning | - | - | 🚧 暂无 |
| electric_meter | - | - | 🚧 暂无 |
| water_leak | - | - | 🚧 暂无 |
| cabinet_header | - | - | 🚧 暂无 |

## 快速开始

### 环境要求

- **TinyGo 0.40+** (用于编译 WASM 驱动)
- **Go 1.21+** (用于网关主程序)

### 安装 TinyGo

```bash
# macOS
brew install tinygo

# Linux
wget https://github.com/tinygo-org/tinygo/releases/download/v0.40.1/tinygo0.40.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf tinygo0.40.1.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/tinygo/bin
```

### 编译所有驱动

```bash
cd drvs
make all
```

### 编译特定驱动

```bash
# 只编译 UPS 驱动
make ups

# 只编译温湿度驱动
make temperature_humidity
```

### 安装到 drivers 目录

```bash
make install
```

## 驱动开发

参考各驱动目录的 README：

- [UPS 驱动文档](ups/README.md)
- [温湿度驱动文档](temperature_humidity/README.md)

## 相关文档

- [Extism 文档](https://extism.org/)
- [TinyGo 文档](https://tinygo.org/)
- [Modbus 协议规范](https://modbus.org/)
