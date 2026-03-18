# Rust Drivers

这里存放基于 Rust 构建的 Wasm 驱动。

- 当前示例目录位于 `rust/poc/`
- 每个驱动按独立 Cargo 工程组织
- 统一构建入口：
  `make -f drvs/Makefile rust`
