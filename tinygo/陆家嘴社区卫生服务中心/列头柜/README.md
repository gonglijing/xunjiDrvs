# 列头柜 Modbus TCP 驱动

## 设备信息

- 设备类型：列头柜
- 协议类型：Modbus TCP
- 功能码：`0x03`（`HOLDING_REGISTER`）
- 驱动文件：`rack_cabinet.go`
- 产物文件：`rack_cabinet.wasm`

## 点表概览

### 电压

- `UA1` `UB1` `UC1` `Uups`（`275~278`，表达式 `v/10`）

### 电流

- `MainsACurr`~`UpsPdu7Curr`（`503~523`，表达式 `v/10`）

### 功率

- `MainsPA`~`UpsPdu7P`（`621~637`，表达式 `v/10`）

### 电能（INIT32）

- `MainsEPA` `MainsEPB` `MainsEPC`
- `MainsPdu1EP`~`MainsPdu7EP`
- 对应地址分布在 `848~873`，双寄存器组合后按 `v/10` 计算

### 开关状态

- `MSS`、`MainsPdu1Switch`~`MainsPdu7Switch`、`UpsPdu1Switch`~`UpsPdu7Switch`
- 地址 `170~186`，表达式 `bitand(v,32768)`（代码实现为 `v & 0x8000`）

## 寄存器读取分组

- 开关段：`170~186`
- 电压段：`275~278`
- 电流段：`503~523`
- 功率段：`621~637`
- 电能段：`848~873`

## 返回示例 JSON

```json
{
  "success": true,
  "points": [
    {"field_name": "UA1", "value": "220.6", "rw": "R", "unit": "V", "label": "市电总输入A"},
    {"field_name": "MainsACurr", "value": "12.5", "rw": "R", "unit": "A", "label": "市电输入A相电流"},
    {"field_name": "MainsPA", "value": "3.8", "rw": "R", "unit": "kW", "label": "市电输出A相功率"},
    {"field_name": "MainsEPA", "value": "1245.7", "rw": "R", "unit": "kWh", "label": "市电输出A相电能"},
    {"field_name": "MSS", "value": "32768", "rw": "R", "unit": "", "label": "市电总输入开关状态"}
  ]
}
```

## 编译

```bash
cd drvs/陆家嘴社区卫生服务中心/列头柜
make rack_cabinet.wasm
```

## 网关配置建议

- `device_address`：设备地址（默认 `1`）
- 资源配置：目标设备 `IP:Port`（Modbus TCP 常用端口 `502`）
- 排障建议：确认网络可达后再开启采集
