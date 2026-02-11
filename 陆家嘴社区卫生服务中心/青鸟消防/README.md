# 青鸟消防主机 Modbus RTU 驱动

## 设备信息

- 设备类型：青鸟消防主机
- 协议类型：Modbus RTU
- 功能码：`0x03`（`HOLDING_REGISTER`）
- 协议来源：`/Users/mac/workspace/pandav2/gateway/deploy/pandax230.db`
- 设备标识：`did=t8AyvifdGo`，共 `275` 个测点
- 驱动文件：`qingniao_fire.go`
- 产物文件：`qingniao_fire.wasm`

## 点表概览

说明：点位从数据库自动生成（`device_tsls` 表），以下仅展示部分样例。

| 属性名 | 属性标识 | 寄存器地址 | 寄存器数量 | 表达式 | 读写 |
|---|---|---:|---:|---|---|
| 1层心电烟感 | `yg0101` | 257 | 1 | `v` | R |
| 1层心电烟感 | `yg0102` | 258 | 1 | `v` | R |
| 1层外科烟感 | `yg0103` | 259 | 1 | `v` | R |
| 1层走道手报 | `sb0133` | 307 | 1 | `v` | R |
| 1层候诊室烟感 | `yg0179` | 377 | 1 | `v` | R |
| 4层口腔烟感 | `yg0229` | 553 | 1 | `v` | R |
| 3层走道声光 | `sg0273` | 627 | 1 | `v` | R |

## 寄存器读取分组

- 连续段A：`257~416`（按 Modbus 上限拆分为 `257+125`、`382+35`）
- 连续段B：`513~627`（`513+115`）
- 共 3 次批量读取覆盖全部 `275` 点

## 返回示例 JSON

```json
{
  "success": true,
  "points": [
    {"field_name": "yj0112", "value": "0", "rw": "R", "unit": "", "label": "1层走道烟感"},
    {"field_name": "sb0132", "value": "0", "rw": "R", "unit": "", "label": "1层走道手报"},
    {"field_name": "lszs0169", "value": "0", "rw": "R", "unit": "", "label": "2层水流指示"}
  ]
}
```

## 编译

```bash
cd drvs/陆家嘴社区卫生服务中心/青鸟消防
make qingniao_fire.wasm
```

## 网关配置建议

- `device_address`：设备从站地址（默认 `1`）
- 串口参数：以数据库 `devices.define` 为准（9600,8,N,1）
- 排障建议：配置 `debug=true` 查看 RTU 收发帧日志
