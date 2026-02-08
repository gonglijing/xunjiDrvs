# UPS Driver

## 简介

UPS（不间断电源）设备驱动目录。

## 状态

| 驱动名称 | 协议 | 状态 |
|---------|------|------|
| ups_kstar | Modbus TCP | ✅ 已实现 |

## 已实现驱动

### ups_kstar

科士达 UPS Modbus TCP 驱动。

#### 功能特性

- 电池参数监测（容量、剩余时间）
- 输入参数监测（频率、各相电压）
- 输出参数监测（频率、各相电压、负载率）

#### 点表

| 字段 | 地址 | 长度 | 缩放 | 小数位 | 单位 | 说明 |
|------|------|------|------|--------|------|------|
| battery_capacity | 100 | 1 | 0.1 | 1 | % | 电池容量 |
| battery_remain_time | 101 | 1 | 1 | 0 | min | 电池剩余时间 |
| input_frequency | 109 | 1 | 0.1 | 1 | Hz | 输入频率 |
| input_voltage_r | 110 | 1 | 0.1 | 1 | V | R相输入电压 |
| input_voltage_s | 111 | 1 | 0.1 | 1 | V | S相输入电压 |
| input_voltage_t | 112 | 1 | 0.1 | 1 | V | T相输入电压 |
| output_frequency | 119 | 1 | 0.1 | 1 | Hz | 输出频率 |
| output_voltage_r | 120 | 1 | 0.1 | 1 | V | R相输出电压 |
| output_voltage_s | 121 | 1 | 0.1 | 1 | V | S相输出电压 |
| output_voltage_t | 122 | 1 | 0.1 | 1 | V | T相输出电压 |
| load_percent_r | 123 | 1 | 1 | 0 | % | R相负载率 |
| load_percent_s | 124 | 1 | 1 | 0 | % | S相负载率 |
| load_percent_t | 125 | 1 | 1 | 0 | % | T相负载率 |

#### 编译

```bash
cd drvs/ups
make ups_kstar.wasm
# 或
tinygo build -o ups_kstar.wasm -target=wasip1 -buildmode=c-shared -opt=z ups_kstar.go
```

#### 相关文档

- [详细驱动文档 README_ups_kstar.md](README_ups_kstar.md)
- [FSU 驱动开发指南](../README.md)
