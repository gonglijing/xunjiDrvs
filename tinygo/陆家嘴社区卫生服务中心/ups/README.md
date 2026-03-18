# 科士达 UPS Modbus TCP 驱动

## 设备信息

- 设备类型：科士达 UPS
- 协议类型：Modbus TCP
- 功能码：`0x03`（`HOLDING_REGISTER`）
- 驱动文件：`ups_kstar.go`
- 产物文件：`ups_kstar.wasm`

## 点表概览

| 属性名 | 属性标识 | 寄存器地址 | 寄存器数量 | 小数位 | 表达式 | 读写 |
|---|---|---:|---:|---:|---|---|
| R相输出电压 | `OUR` | 120 | 1 | 1 | `v/10` | R |
| S相输出电压 | `OUS` | 121 | 1 | 1 | `v/10` | R |
| T相输出电压 | `OUT` | 122 | 1 | 1 | `v/10` | R |
| 输出频率 | `OH` | 119 | 1 | 1 | `v/10` | R |
| R相负载率 | `loadR` | 123 | 1 | 0 | `v` | R |
| S相负载率 | `loadS` | 124 | 1 | 0 | `v` | R |
| T相负载率 | `loadT` | 125 | 1 | 0 | `v` | R |
| R相输入电压 | `IUR` | 110 | 1 | 1 | `v/10` | R |
| S相输入电压 | `IUS` | 111 | 1 | 1 | `v/10` | R |
| T相输入电压 | `IOT` | 112 | 1 | 1 | `v/10` | R |
| 输入频率 | `IH` | 109 | 1 | 1 | `v/10` | R |
| 电池容量 | `qos` | 100 | 1 | 1 | `v/10` | R |
| 电池剩余时间 | `ltime` | 101 | 1 | 0 | `v` | R |

## 寄存器读取分组

- 输出段：`119~125`（读取 `OH`、`OUR`、`OUS`、`OUT`、`loadR`、`loadS`、`loadT`）
- 输入段：`109~112`（读取 `IH`、`IUR`、`IUS`、`IOT`）
- 电池段：`100~101`（读取 `qos`、`ltime`）

## 返回示例 JSON

```json
{
  "success": true,
  "points": [
    {"field_name": "OUR", "value": "220.1", "rw": "R", "unit": "V", "label": "R相输出电压"},
    {"field_name": "OH", "value": "50.0", "rw": "R", "unit": "Hz", "label": "输出频率"},
    {"field_name": "IUR", "value": "219.8", "rw": "R", "unit": "V", "label": "R相输入电压"},
    {"field_name": "qos", "value": "95.0", "rw": "R", "unit": "%", "label": "电池容量"},
    {"field_name": "ltime", "value": "87", "rw": "R", "unit": "min", "label": "电池剩余时间"}
  ]
}
```

## 编译

```bash
cd drvs/陆家嘴卫生活动中心/ups
make ups_kstar.wasm
```

## 网关配置建议

- `device_address`：设备地址（默认 `1`）
- 资源配置：目标设备 `IP:Port`（Modbus TCP 常用端口 `502`）
- 排障建议：确认网络可达后再开启采集
