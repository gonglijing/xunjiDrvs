# Rust Modbus RTU 驱动模板

这是一个可以直接编译成 `WASM` 的 Rust 驱动模板，目标是和当前 `TinyGo` 驱动保持同一类宿主接口约定：

- 导出 `handle`、`describe`、`version`
- 导入宿主提供的 `serial_transceive`
- 输入使用网关传入的 `config`
- 输出保持统一的 `success/productKey/points/error` 结构
- 支持 `func_name=write` 的单点单寄存器写入范式

## 适用场景

- 你想把现有 TinyGo 串口驱动迁移到 Rust
- 你想验证 “Rust 是否也能一次编译，多架构运行”
- 你需要一个可读性优先、容易改点表的基线工程

这里的答案是：可以，但前提是编译成 `WASM` 插件，而不是本地原生二进制。

## 目录说明

- `src/lib.rs`：模板核心代码
- `Cargo.toml`：Rust 依赖与发布优化配置
- `Makefile`：统一构建入口

## 构建

```bash
make -C drvs/rust/template/rust_modbus_rtu_template all
```

产物：

```bash
drvs/rust/template/rust_modbus_rtu_template/rust_modbus_rtu_template.wasm
```

## 你通常只需要改这几处

1. `DRIVER_VERSION`
2. `DRIVER_PRODUCT_KEY`
3. `FUNC_CODE_READ`
4. `POINT_CONFIGS`
5. `read_all_points` 里的换算逻辑
6. 如需写入，再替换 `write_single_point` 的字段映射规则

## 当前模板的默认点表

- `temperature`：地址 `0`，长度 `1`，`v/10`，默认 `RW`
- `humidity`：地址 `1`，长度 `1`，`v/10`，默认 `RW`
- `dewtemperature`：地址 `2`，长度 `1`，`v/10`

这只是演示点表，方便你对照 TinyGo 版共济温湿度驱动理解迁移方式。

## 写入约定

模板已经内置“单次调用，只写一个字段”的参考实现，对齐当前网关与 TinyGo 驱动的约定：

- `config.func_name=write`
- `config.field_name=<字段名>`
- `config.value=<工程量字符串>`

默认情况下：

- `temperature`
- `humidity`

会被视为可写字段，并通过标准 Modbus RTU `0x06` 单寄存器写下发。

如果你迁移的真实设备不是单寄存器写，通常需要替换：

1. `FUNC_CODE_WRITE_SINGLE`
2. `build_write_single_frame`
3. `parse_write_single_response`
4. `encode_write_value`
5. `write_single_point`

## 运行前提

“一次编译，到处运行”成立的前提不是 Rust 语言本身，而是：

- 编译目标仍然是同一种 `WASM`
- 宿主运行时兼容该 `WASM`
- 宿主继续提供同名同签名的 `serial_transceive`
- 插件导出函数和 JSON 协议保持一致

如果改成原生 `so`、`dll`、`exe`，那就重新受到 CPU 架构和操作系统影响。
