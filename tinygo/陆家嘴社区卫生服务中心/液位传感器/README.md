# 液位传感器 Modbus RTU 驱动

## 设备信息

- 设备类型：液位传感器
- 协议类型：Modbus RTU
- 功能码：`0x03`（`HOLDING_REGISTER`）
- 驱动文件：`level.go`
- 产物文件：`level.wasm`

## 点表概览

| 属性名 | 属性标识 | 寄存器地址 | 寄存器数量 | 小数位 | 表达式 | 读写 |
|---|---|---:|---:|---:|---|---|
| 液位 | `level` | 0 | 2 | 3 | `(v-101665)/9800` | R |
| 温度 | `wtemp` | 2 | 1 | 2 | `v/100` | R |

## 寄存器读取分组

- 连续读取段：`0~2`
- `level` 使用 2 个寄存器拼接后再换算
- `wtemp` 使用 1 个寄存器直接换算

## 写入支持

当前驱动为只读驱动。

- 返回点位的 `rw` 全部为 `R`
- 如果传入 `config.func_name=write`，驱动会明确返回“不支持写入”
- FSU 侧应按只读测点处理

## 返回示例 JSON

```json
{
  "success": true,
  "productKey": "ljzchc_level_sensor",
  "points": [
    {"field_name": "level", "value": "1.245", "rw": "R", "unit": "", "label": "液位"},
    {"field_name": "wtemp", "value": "26.35", "rw": "R", "unit": "°C", "label": "温度"}
  ]
}
```

## 编译

```bash
cd /Users/mac/workspace/xunji/fsu/drvs/tinygo/陆家嘴社区卫生服务中心/液位传感器
make level.wasm
```

## 网关配置建议

- `device_address`：设备从站地址，默认 `1`
- 串口参数：按现场设备一致配置
- 排障建议：可开启 `debug=true` 查看收发帧
