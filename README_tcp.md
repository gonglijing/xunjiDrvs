# 温湿度驱动（Modbus TCP 版）

## 功能概览
- 读温度/湿度：寄存器 0x0000~0x0001，缩放 0.1。
- 读温度告警阈值：寄存器 0x0022。
- 写设备地址：寄存器 0x0021（0x06）。
- 写温度告警阈值：寄存器 0x0022（0x06）。
- 元数据：`describe` 返回可写字段 `device_addr`、`temp_alarm_threshold`。

## Host 依赖（网关提供）
- `tcp_transceive`
- `output`

## 导出接口（插件）
- `handle`：入口；`func_name=read|write`。
- `describe`：可写字段列表。

## 输入（宿主传入 envelope.config）
```json
{
  "device_address": 1,
  "func_name": "read",          // 或 "write"
  "field_name": "device_addr",
  "value": 2                     // 写时使用
}
```

## 返回
同 RTU 版：`success/data.points` 列表；失败返回 `success=false` 与 `error`。

## 构建
```bash
cd drvs
tinygo build -o th_modbustcp.wasm -target=wasi -stack-size=64k th_modbustcp.go
```

## 实现要点（th_modbustcp.go）
- 读：`tcp_transceive` 发送 0x03 帧（MBAP，无 CRC），一次读温湿度，再读阈值；校验功能码、字节数。
- 写：`field_name` 路由到 0x0021/0x0022，构造 0x06 帧（MBAP），校验回显。
- 配置：`getConfig` 解析宿主输入 `config`，默认 func_name=read。

## 需客户自行实现/可定制
- 设备寄存器、缩放、功能码与协议差异需按实际设备修改封包/解码逻辑。
- 可写字段元数据可根据设备能力扩展或调整描述。
- 可增加重试、异常日志、范围校验等健壮性处理。
