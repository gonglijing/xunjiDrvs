# msj 温湿度 Modbus RTU 驱动

## 设备信息

- 设备类型：msj 温湿度
- 协议类型：Modbus RTU
- 功能码：`0x03`（`HOLDING_REGISTER`）
- 驱动文件：`msj_th.go`
- 产物文件：`msj_th.wasm`

## 点表概览

| 属性名 | 属性标识 | 寄存器地址 | 寄存器数量 | 小数位 | 表达式 | 读写 |
|---|---|---:|---:|---:|---|---|
| 温度 | `temperature` | 0 | 1 | 1 | `v/10` | R |
| 湿度 | `humidity` | 1 | 1 | 1 | `v/10` | R |

## 寄存器读取分组

- 批量读取：`0~1`
- 映射顺序：`temperature` → `humidity`

## 写入支持

当前驱动为只读驱动。

- 返回点位的 `rw` 只有 `R`
- FSU 侧不应把它展示为可写点
- 如果传入 `config.func_name=write`，驱动会明确返回“不支持写入”

## 返回示例 JSON

```json
{
  "success": true,
  "productKey": "ljzchc_msj_th",
  "points": [
    {"field_name": "temperature", "value": "26.3", "rw": "R", "unit": "℃", "label": "温度"},
    {"field_name": "humidity", "value": "58.4", "rw": "R", "unit": "%", "label": "湿度"}
  ]
}
```

## 编译

```bash
cd /Users/mac/workspace/xunji/fsu/drvs/tinygo/陆家嘴社区卫生服务中心/msj
make msj_th.wasm
```

## 网关配置建议

- `device_address`：设备从站地址，默认 `1`
- 串口参数：按现场设备一致配置
- 排障建议：可开启 `debug=true` 查看收发帧
