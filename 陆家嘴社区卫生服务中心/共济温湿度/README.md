# 共济温湿度 Modbus RTU 驱动

## 设备信息

- 设备类型：共济温湿度
- 协议类型：Modbus RTU
- 功能码：`0x04`（`INPUT_REGISTER`）
- 驱动文件：`gongji_th.go`
- 产物文件：`gongji_th.wasm`

## 点表概览

| 属性名 | 属性标识 | 寄存器地址 | 寄存器数量 | 小数位 | 表达式 | 读写 |
|---|---|---:|---:|---:|---|---|
| 温度 | `temperature` | 0 | 1 | 1 | `v/10` | R |
| 湿度 | `humidity` | 1 | 1 | 1 | `v/10` | R |
| 漏点温度 | `dewtemperature` | 2 | 1 | 1 | `v/10` | R |

## 寄存器读取分组

- 批量读取：`0~2`（共 3 个寄存器）
- 映射顺序：`temperature` → `humidity` → `dewtemperature`

## 返回示例 JSON

```json
{
  "success": true,
  "points": [
    {"field_name": "temperature", "value": "26.3", "rw": "R", "unit": "℃", "label": "温度"},
    {"field_name": "humidity", "value": "58.4", "rw": "R", "unit": "%", "label": "湿度"},
    {"field_name": "dewtemperature", "value": "18.7", "rw": "R", "unit": "℃", "label": "漏点温度"}
  ]
}
```

## 编译

```bash
cd drvs/陆家嘴社区卫生服务中心/共济温湿度
make gongji_th.wasm
```

## 网关配置建议

- `device_address`：设备从站地址（默认 `1`）
- 串口参数：按现场设备一致配置（波特率/数据位/校验/停止位）
- 排障建议：可开启 `debug=true` 查看收发帧
