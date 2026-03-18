# 共济温湿度 Rust 版驱动

这是对现有 TinyGo 驱动
[drvs/tinygo/陆家嘴社区卫生服务中心/共济温湿度/gongji_th.go](/Users/mac/workspace/xunji/fsu/drvs/tinygo/陆家嘴社区卫生服务中心/共济温湿度/gongji_th.go)
的等价 Rust 实现。

目标：

- 验证现有宿主 ABI 可以直接运行 Rust 版真实协议驱动
- 保持与 TinyGo 版本一致的点表、功能码、默认配置和输出字段

## 协议信息

- 协议：Modbus RTU
- 功能码：`0x04`
- 产物：`gongji_th_rust.wasm`
- 产品标识：`ljzchc_gongji_th`

## 点表

- `temperature`：地址 `0`，长度 `1`，`v/10`
- `humidity`：地址 `1`，长度 `1`，`v/10`
- `dewtemperature`：地址 `2`，长度 `1`，`v/10`

## 构建

```bash
make -C drvs/rust/poc/rust_gongji_th all
```

## 测试

```bash
cargo test
```

## 输入配置

- `device_address`：从站地址，默认 `1`
- `debug`：`true/false`，可选
- `timeout_ms`：串口超时，默认 `1000`

## 备注

- 这里仍然直接调用宿主的 `serial_transceive`
- 如果这个版本要真正替换线上 TinyGo 驱动，建议再补一轮和现场样本帧的对拍测试
