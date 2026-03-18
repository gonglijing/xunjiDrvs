# 压力传感器 Modbus RTU 驱动

## 设备信息

- 设备类型：压力传感器
- 协议类型：Modbus RTU
- 功能码：`0x03`（`HOLDING_REGISTER`）
- 驱动文件：`pressure.go`
- 产物文件：`pressure.wasm`

## 点表概览

| 属性名 | 属性标识 | 寄存器地址 | 寄存器数量 | 小数位 | 表达式 | 读写 |
|---|---|---:|---:|---:|---|---|
| 压力 | `p` | 4 | 1 | 0 | `v/1000` | R |

## 寄存器读取分组

- 单点读取段：`4~4`

## 写入支持

当前驱动为只读驱动。

- 返回点位的 `rw` 为 `R`
- 如果传入 `config.func_name=write`，驱动会明确返回“不支持写入”
- FSU 侧应按只读测点处理

## 返回示例 JSON

```json
{
  "success": true,
  "productKey": "ljzchc_pressure_sensor",
  "points": [
    {"field_name": "p", "value": "12", "rw": "R", "unit": "", "label": "压力"}
  ]
}
```

## 编译

```bash
cd /Users/mac/workspace/xunji/fsu/drvs/tinygo/陆家嘴社区卫生服务中心/压力传感器
make pressure.wasm
```

## 网关配置建议

- `device_address`：设备从站地址，默认 `1`
- 串口参数：按现场设备一致配置
- 排障建议：可开启 `debug=true` 查看收发帧
