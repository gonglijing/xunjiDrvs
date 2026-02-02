# 温湿度驱动（Modbus RTU 版）

## 功能概览
- 读温度/湿度：寄存器 0x0000~0x0001，缩放 0.1。
- 读温度告警阈值：寄存器 0x0022。
- 写设备地址：寄存器 0x0021（0x06）。
- 写温度告警阈值：寄存器 0x0022（0x06）。
- 元数据：`describe` 返回可写字段 `device_addr`、`temp_alarm_threshold`。

## Host 依赖（网关提供）
- `serial_transceive`
- `output`

## 导出接口（插件）
- `handle`：入口；`func_name=read|write`。
- `describe`：可写字段列表。

## 输入（宿主传入 envelope.config）
```json
{
  "device_address": 1,
  "func_name": "read",          // 或 "write"
  "field_name": "temp_alarm_threshold",
  "value": 300                   // 写时使用
}
```

## 返回
成功：
```json
{
  "success": true,
  "data": {"points": [
    {"field_name":"temperature","value":25.3,"rw":"R"},
    {"field_name":"humidity","value":60.5,"rw":"R"},
    {"field_name":"temp_alarm_threshold","value":30,"rw":"RW"}
  ]}
}
```
失败：`{"success":false,"error":"..."}`

## 构建
```bash
cd drvs
tinygo build -o th_modbusrtu.wasm -target=wasi -stack-size=64k th_modbusrtu.go
```

## 实现要点（th_modbusrtu.go）
- 读：`serial_transceive` 发送 0x03 帧读 0x0000~0x0001，再读 0x0022；CRC16 校验。
- 写：`field_name` 路由到 0x0021 或 0x0022，构造 0x06 帧，校验回显 + CRC。
- 配置：`getConfig` 解析宿主输入 `config`，有默认兜底。

## 需客户自行实现/可定制
- 如设备寄存器/比例/功能码不同，请修改寄存器地址、缩放、帧封装和解码逻辑。
- 可写字段可按设备能力扩展或调整描述文本。
- 如需更多错误/范围校验，可在写路径补充。
