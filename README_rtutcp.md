# 温湿度驱动示例文档（RTU/TCP 分版）

本文档描述两份示例驱动：`th_modbusrtu.wasm`（串口 RTU）与 `th_modbustcp.wasm`（网络 TCP）。二者功能一致：
- 读温度/湿度：寄存器 0x0000~0x0001，缩放 0.1。
- 读温度告警阈值：寄存器 0x0022。
- 写设备地址：寄存器 0x0021（功能码 0x06）。
- 写温度告警阈值：寄存器 0x0022（功能码 0x06）。
- 元数据 `describe` 提供可写字段列表（不暴露寄存器细节）。

## Host 依赖
- RTU 版：`serial_transceive`, `output`（由网关提供）。
- TCP 版：`tcp_transceive`, `output`（由网关提供）。

## 导出接口
- `handle`：统一入口，`func_name` 路由 `read` / `write`。
- `describe`：返回可写字段元数据：`device_addr`、`temp_alarm_threshold`。

## 输入配置（网关传入的 JSON envelope.config）
```json
{
  "device_address": 1,
  "func_name": "read",          // 或 "write"
  "field_name": "temp_alarm_threshold", // 写时必填，读时可空
  "value": 30,                   // 写时的值
  "protocol": "rtu" | "tcp"   // 仅 RTU 版可忽略，TCP 版默认 tcp
}
```

## 返回格式
```json
{
  "success": true,
  "data": {
    "points": [
      {"field_name":"temperature", "value":25.3, "rw":"R"},
      {"field_name":"humidity", "value":60.5, "rw":"R"},
      {"field_name":"temp_alarm_threshold", "value":30, "rw":"RW"}
    ]
  }
}
```
失败时：`{"success":false,"error":"..."}`。

## 构建
```bash
# RTU
cd drvs
tinygo build -o th_modbusrtu.wasm -target=wasi -stack-size=64k th_modbusrtu.go
# TCP
tinygo build -o th_modbustcp.wasm -target=wasi -stack-size=64k th_modbustcp.go
```

## RTU 版实现要点（th_modbusrtu.go）
- 读：使用 `serial_transceive` 发送 0x03 帧，一次读 0x0000~0x0001；再读 0x0022。
- 写：`field_name` 路由到寄存器，组 0x06 帧并回显校验。
- CRC16 校验，RTU 帧自带 CRC。

## TCP 版实现要点（th_modbustcp.go）
- 读：使用 `tcp_transceive` 发送 0x03 帧（MBAP，无 CRC），一次读温湿度，再读阈值。
- 写：`field_name` 路由 0x0021 / 0x0022，构造 0x06 帧（MBAP，无 CRC）。
- 响应校验基于 MBAP/功能码/字节数。

## 可写字段（describe 返回）
```json
{
  "success": true,
  "data": {
    "writable": [
      {"field": "device_addr", "label": "设备地址", "desc": "写入新的设备地址"},
      {"field": "temp_alarm_threshold", "label": "温度告警阈值", "desc": "设置温度告警阈值"}
    ]
  }
}
```

## 调用示例（网关 /api/devices/{id}/execute）
- 读：`{ "function":"handle", "params": { "func_name":"read" } }`
- 写阈值：`{ "function":"handle", "params": { "func_name":"write", "field":"temp_alarm_threshold", "value": 300 } }`
- 写地址：`{ "function":"handle", "params": { "func_name":"write", "field":"device_addr", "value": 2 } }`

## 注意
- 网关需为 RTU 资源绑定串口到 `resource_id`，TCP 资源绑定 TCP 连接，驱动内部不感知资源 ID。
- 驱动内部未做单位/范围校验，前端可结合元数据自行限制。
