# Rust Drivers

这里存放基于 Rust 构建的 Wasm 驱动。

- 当前正式目录位于 `rust/template/`
- 每个驱动按独立 Cargo 工程组织
- 统一构建入口：
  `make -f drvs/Makefile rust`

## 推荐起点

如果要新建一个 Rust 串口驱动，优先从下面这个模板复制：

- [rust_modbus_rtu_template](/Users/mac/workspace/xunji/fsu/drvs/rust/template/rust_modbus_rtu_template)

这个模板已经包含：

- `handle` / `describe` / `version`
- `serial_transceive` 宿主导入
- 标准 Modbus RTU 读寄存器请求与 CRC 校验
- 标准 Modbus RTU `0x06` 单寄存器写请求与响应校验
- 点表驱动的输出拼装方式
- `func_name=write` 的单点写入入口
- 基本单元测试
