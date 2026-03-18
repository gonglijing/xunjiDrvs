# FSU Device Drivers

`drvs` 是 FSU（场站单元）侧的设备驱动目录，采用 `Extism + WebAssembly` 插件架构。

当前驱动分为两类：

- `TinyGo`：现网驱动主体，目录位于 `drvs/tinygo/`
- `Rust`：模板化驱动目录，当前位于 `drvs/rust/template/`

## 当前目录结构

```text
drvs/
├── Makefile
├── README.md
├── tinygo/
│   ├── go.mod
│   ├── go.sum
│   ├── README.md
│   ├── pkg/
│   └── 陆家嘴社区卫生服务中心/
│       ├── msj/
│       ├── ups/
│       ├── 共济温湿度/
│       ├── 列头柜/
│       ├── 压力传感器/
│       ├── 液位传感器/
│       ├── 美的空调/
│       ├── 青鸟消防/
│       └── 高特电池网关/
└── rust/
    ├── README.md
    └── template/
        └── rust_modbus_rtu_template/
```

## 驱动清单

| 语言 | 站点/分组 | 目录 | 协议 | 写入支持 |
|---|---|---|---|---|
| TinyGo | 陆家嘴社区卫生服务中心 | `msj` | Modbus RTU | 否 |
| TinyGo | 陆家嘴社区卫生服务中心 | `ups` | Modbus TCP | 否 |
| TinyGo | 陆家嘴社区卫生服务中心 | `共济温湿度` | Modbus RTU | 否 |
| TinyGo | 陆家嘴社区卫生服务中心 | `列头柜` | Modbus TCP | 否 |
| TinyGo | 陆家嘴社区卫生服务中心 | `压力传感器` | Modbus RTU | 否 |
| TinyGo | 陆家嘴社区卫生服务中心 | `液位传感器` | Modbus RTU | 否 |
| TinyGo | 陆家嘴社区卫生服务中心 | `美的空调` | Modbus RTU | 是 |
| TinyGo | 陆家嘴社区卫生服务中心 | `青鸟消防` | Modbus RTU | 否 |
| TinyGo | 陆家嘴社区卫生服务中心 | `高特电池网关` | Modbus RTU | 否 |
| Rust | `template` | `rust_modbus_rtu_template` | Modbus RTU Template | 是，模板示例 |

## 统一读写约定

所有驱动都走统一的 `config` 输入结构。

读取时：

- `config.func_name` 缺省视为 `read`
- 返回结果中的 `points[].rw` 表示点位读写属性
- `rw` 中包含 `W`，表示该点位允许 FSU 发起写入

写入时：

- 一次调用只写一个点
- FSU 通过 `field_name + value` 指定目标点和值
- 具体寄存器映射、数值编码、协议校验由驱动内部实现

参考写入请求：

```json
{
  "config": {
    "func_name": "write",
    "field_name": "TEMSET",
    "value": "24.0",
    "device_address": "1"
  }
}
```

说明：

- `field_name` 必须对应驱动返回点表中的字段标识
- `value` 使用工程量字符串，由驱动自行换算为寄存器值
- 当前 TinyGo 现网驱动里，只有 `美的空调` 实现了真实写入
- 其余 TinyGo 驱动收到 `func_name=write` 时会显式返回“不支持写入”

## 编译说明

### 1. 编译全部驱动

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile all
```

### 2. 仅编译 TinyGo 或 Rust 驱动

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile tinygo
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile rust
```

### 3. 编译单个驱动

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/tinygo/陆家嘴社区卫生服务中心/美的空调/Makefile all
make -f /Users/mac/workspace/xunji/fsu/drvs/rust/template/rust_modbus_rtu_template/Makefile all
```

### 4. 安装到网关 `drivers` 目录

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile install
```

## 环境要求

- `TinyGo 0.40+`
- `Rust stable + cargo`
- `Go 1.21+`

## 开发约定

- TinyGo 模块根目录为 `drvs/tinygo/`
- TinyGo 共享包位于 `drvs/tinygo/pkg/`
- Rust 驱动按独立 Cargo 工程组织
- 驱动产物 `*.wasm` 直接生成在各自协议目录下
- 协议变更优先更新 `points.xlsx`，再同步代码与 README
- 文档中如果声明点位为 `RW`，必须与实际驱动实现保持一致

## 相关文档

- [Extism 文档](https://extism.org/)
- [TinyGo 文档](https://tinygo.org/)
- [Rust 文档](https://www.rust-lang.org/)
- [Modbus 协议规范](https://modbus.org/)
