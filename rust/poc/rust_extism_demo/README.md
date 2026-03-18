# Rust Modbus RTU Template

这是一个可直接复制的 Rust + Extism + Modbus RTU 驱动模板。

目标：

- 验证 Rust 可以直接接入当前 FSU 的 Wasm 驱动 ABI
- 提供一份“真实 RTU 只读驱动”的最小模板
- 把协议开发时真正会反复改的部分收敛到少数几个区域

## 构建

```bash
make -C drvs/rust/poc/rust_extism_demo all
```

产物：

```text
drvs/rust/poc/rust_extism_demo/rust_extism_demo.wasm
```

## 宿主输入

插件按当前网关宿主的 JSON 输入协议解析。典型输入：

```json
{
  "device_id": 1,
  "device_name": "demo",
  "resource_id": 2,
  "resource_type": "serial",
  "config": {
    "device_address": "1",
    "timeout_ms": "1000",
    "debug": "true"
  },
  "device_config": ""
}
```

## 行为

- 读取 `device_address`
- 基于模板点表批量构造 Modbus RTU 读帧
- 调用宿主 `serial_transceive`
- 校验 CRC 并解析寄存器
- 返回宿主兼容的 `points` JSON

## 说明

- 这里继续使用原始 Wasm import：
  `#[link(wasm_import_module = "extism:host/user")]`
- 这样可以直接贴合当前宿主已经实现的 `serial_transceive(i64, i64, i64, i64, i64) -> i64`
- 这个模板默认是“只读 RTU”：
  你主要需要改的是 `POINT_CONFIGS`、`FUNC_CODE_READ_INPUT`、`scale/decimals`

## 你通常只需要改这几处

1. `DRIVER_PRODUCT_KEY`
2. `POINT_CONFIGS`
3. `FUNC_CODE_READ_INPUT` 或寄存器读逻辑
4. `read_all_points` 里对寄存器值的转换方式

## 构建

```bash
make -C drvs/rust/poc/rust_extism_demo all
```

## 测试

```bash
cargo test
```
