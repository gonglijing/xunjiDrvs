# 美的空调 Modbus RTU 驱动

## 设备信息

- 设备类型：美的空调
- 协议类型：Modbus RTU
- 功能码：`0x03`（`HOLDING_REGISTER`）
- 驱动文件：`midea_ac.go`
- 产物文件：`midea_ac.wasm`

## 点表概览

| 属性名 | 属性标识 | 寄存器地址 | 寄存器数量 | 小数位 | 表达式 | 读写 |
|---|---|---:|---:|---:|---|---|
| 温度设点 | `TEMSET` | 0 | 1 | 1 | `v/10` | R |
| 湿度设点 | `HUMSET` | 2 | 1 | 1 | `v/10` | R |
| 环境温度 | `TEM` | 48 | 1 | 1 | `v/10` | R |
| 环境湿度 | `HUM` | 49 | 1 | 1 | `v/10` | R |
| 室内高温报警值 | `IHTAV` | 17 | 1 | 1 | `v/10` | R |
| 室内低温报警值 | `ILTAV` | 18 | 1 | 1 | `v/10` | R |
| 高湿度报警值 | `HHAV` | 19 | 1 | 1 | `v/10` | R |
| 低湿度报警值 | `LHAV` | 20 | 1 | 1 | `v/10` | R |
| 设备地址 | `ADD` | 94 | 1 | 1 | `v&0x01` | R |

## 寄存器读取分组

- 设点段：`0~2`（读取 `TEMSET`、`HUMSET`）
- 报警阈值段：`17~20`（读取 `IHTAV`、`ILTAV`、`HHAV`、`LHAV`）
- 环境段：`48~49`（读取 `TEM`、`HUM`）
- 地址段：`94`（读取 `ADD`，按 `v & 0x01` 处理）

## 返回示例 JSON

```json
{
  "success": true,
  "points": [
    {"field_name": "TEMSET", "value": "24.0", "rw": "R", "unit": "℃", "label": "温度设点"},
    {"field_name": "HUMSET", "value": "60.0", "rw": "R", "unit": "%", "label": "湿度设点"},
    {"field_name": "TEM", "value": "26.3", "rw": "R", "unit": "℃", "label": "环境温度"},
    {"field_name": "HUM", "value": "58.4", "rw": "R", "unit": "%", "label": "环境湿度"},
    {"field_name": "ADD", "value": "1.0", "rw": "R", "unit": "", "label": "设备地址"}
  ]
}
```

## 编译

```bash
cd drvs/陆家嘴社区卫生服务中心/美的空调
make midea_ac.wasm
```

## 网关配置建议

- `device_address`：设备从站地址（默认 `1`）
- 串口参数：按现场设备一致配置（波特率/数据位/校验/停止位）
- 排障建议：可开启 `debug=true` 查看收发帧
