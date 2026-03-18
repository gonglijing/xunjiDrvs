# FSU Device Drivers

FSU（场站单元）设备驱动目录，基于 **Extism + WebAssembly** 的插件架构，当前同时支持 **TinyGo** 和 **Rust** 驱动。

## 当前目录结构

```text
drvs/
├── Makefile
├── tinygo/
│   ├── go.mod
│   ├── go.sum
│   ├── pkg/
│   └── 陆家嘴社区卫生服务中心/
│       ├── ups/
│       ├── 共济温湿度/
│       ├── 列头柜/
│       ├── 压力传感器/
│       ├── 液位传感器/
│       ├── 美的空调/
│       ├── 青鸟消防/
│       └── 高特电池网关/
└── rust/
    └── poc/
        ├── rust_extism_demo/
        └── rust_gongji_th/
```

> TinyGo 驱动目录下统一包含：
>
> - Go 模块文件（`go.mod`、`go.sum`）
> - 协议代码（`*.go`）
> - 编译脚本（`Makefile`）
> - 点位表（`points.xlsx`）
> - 协议文档（`README.md`，部分历史目录可能暂缺）
>
> 产物 `*.wasm` 直接生成在协议目录下，不再使用 `build/` 目录。

> Rust 驱动目录按 Cargo 工程组织，示例当前放在 `drvs/rust/poc/`。

## 驱动清单

| 语言 | 站点/分组 | 目录 | 协议 |
|---|---|---|---|
| TinyGo | 陆家嘴社区卫生服务中心 | `ups` | Modbus TCP |
| TinyGo | 陆家嘴社区卫生服务中心 | `共济温湿度` | Modbus RTU |
| TinyGo | 陆家嘴社区卫生服务中心 | `列头柜` | Modbus TCP |
| TinyGo | 陆家嘴社区卫生服务中心 | `压力传感器` | Modbus RTU |
| TinyGo | 陆家嘴社区卫生服务中心 | `液位传感器` | Modbus RTU |
| TinyGo | 陆家嘴社区卫生服务中心 | `美的空调` | Modbus RTU |
| TinyGo | 陆家嘴社区卫生服务中心 | `青鸟消防` | Modbus RTU |
| TinyGo | 陆家嘴社区卫生服务中心 | `高特电池网关` | Modbus RTU |
| Rust | `poc` | `rust_extism_demo` | Modbus RTU Template |
| Rust | `poc` | `rust_gongji_th` | Modbus RTU |

## 编译说明

### 1) 编译全部驱动（推荐）

可在任意目录执行：

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile all
```

### 2) 仅编译 TinyGo 或 Rust 驱动

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile tinygo
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile rust
```

### 3) 编译单个协议驱动

同样支持在任意目录执行：

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/tinygo/陆家嘴社区卫生服务中心/美的空调/Makefile all
make -f /Users/mac/workspace/xunji/fsu/drvs/rust/poc/rust_gongji_th/Makefile all
```

### 4) 安装到网关 drivers 目录

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile install
```

## 环境要求

- **TinyGo 0.40+**（编译 TinyGo WASM 驱动）
- **Rust stable + cargo**（编译 Rust WASM 驱动）
- **Go 1.21+**（网关主程序）

## 协议开发约定

- RTU/TCP 驱动源码按统一注释分区：
  - `【固定不变】`（Host 声明、入口、通信与工具函数）
  - `【用户修改】`（点表定义、寄存器、读取逻辑）
- TinyGo 共享包位于 `drvs/tinygo/pkg/`
- TinyGo 模块根位于 `drvs/tinygo/`
- 协议变更优先更新 `points.xlsx`，再同步代码。

## 相关文档

- [Extism 文档](https://extism.org/)
- [TinyGo 文档](https://tinygo.org/)
- [Rust 文档](https://www.rust-lang.org/)
- [Modbus 协议规范](https://modbus.org/)
