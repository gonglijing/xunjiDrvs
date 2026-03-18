// =============================================================================
// 压力传感器 - Modbus RTU 驱动
// =============================================================================
//
// 设备点表:
//   - 压力(p): FC=03(HOLDING_REGISTER), 地址=0x0004, 长度=1
//     数据类型=int64, 读写=R, 表达式=v/1000
//
// Host 提供: serial_transceive
//
// =============================================================================
package main

import (
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/hostio"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/modbusrtu"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/tinydrv"
)

// =============================================================================
// 【固定不变】Host 函数声明
// =============================================================================
//
//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

func callSerialTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return serial_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

// =============================================================================
// 【固定不变】配置结构（网关传入）
// =============================================================================
type DriverConfig struct {
	DeviceAddress int    `json:"device_address"` // Modbus 从站地址
	FuncName      string `json:"func_name"`      // "read" | "write"
	FieldName     string `json:"field_name"`     // 可写字段名
	Value         string `json:"value"`          // 写操作的值
	Debug         bool   `json:"debug"`          // 调试模式
}

type DriverPoint = tinydrv.Point
type HandleResponse = tinydrv.HandleResponse

type DescribeResponse = tinydrv.DescribeResponse
type VersionData = tinydrv.VersionData
type VersionResponse = tinydrv.VersionResponse
type ErrorResponse = tinydrv.ErrorResponse

// =============================================================================
// 【用户修改】驱动版本
// =============================================================================
const (
	DriverVersion    = "1.0.0"
	DriverProductKey = "ljzchc_pressure_sensor"
)

// =============================================================================
// 【用户修改】点表定义
// =============================================================================
// 根据实际设备修改以下点表配置
const (
	// 寄存器地址定义
	// 格式: RegisterName = Address // 说明
	REG_PRESSURE = 0x0004 // 压力寄存器

	// 功能码定义
	FUNC_CODE_READ = 0x03 // 读保持寄存器
)

// =============================================================================
// 【用户修改】点表配置
// =============================================================================
// 定义所有需要读取的测点
// fields: 字段名, 按实际设备修改
// decimals: 有效小数位数, 按实际设备修改
var pointConfig = []PointConfig{
	{Field: "p", Address: REG_PRESSURE, Length: 1, Scale: 0.001, Decimals: 0, RW: "R", Unit: "", Label: "压力"},
}

// 点表配置结构
type PointConfig struct {
	Field    string  // 字段名
	Address  uint16  // 寄存器地址
	Length   uint16  // 寄存器数量
	Scale    float64 // 缩放系数
	Decimals int     // 有效小数位数
	RW       string  // 读写属性
	Unit     string  // 单位
	Label    string  // 显示标签
}

// =============================================================================
// 【固定不变】驱动入口
// =============================================================================
//
//go:wasmexport handle
func handle() int32 {
	defer func() {
		if r := recover(); r != nil {
			tinydrv.OutputJSON(ErrorResponse{Success: false, Error: "panic"})
		}
	}()

	cfg := getConfig()

	// 读操作 - 读取所有监控参数
	points := readAllPoints(cfg.DeviceAddress, cfg.Debug)

	tinydrv.OutputJSON(HandleResponse{
		Success:    true,
		ProductKey: DriverProductKey,
		Points:     points,
	})
	return 0
}

// =============================================================================
// 【固定不变】描述可写字段
// =============================================================================
//
//go:wasmexport describe
func describe() int32 {
	tinydrv.OutputJSON(DescribeResponse{Success: true})
	return 0
}

// =============================================================================
// 【固定不变】驱动版本
// =============================================================================
//
//go:wasmexport version
func version() int32 {
	tinydrv.OutputJSON(VersionResponse{
		Success: true,
		Data: VersionData{
			Version:    DriverVersion,
			ProductKey: DriverProductKey,
		},
	})
	return 0
}

// =============================================================================
// 【用户修改】读取所有测点
// =============================================================================
// 根据点表配置批量读取寄存器
func readAllPoints(devAddr int, debug bool) []DriverPoint {
	points := make([]DriverPoint, 0, len(pointConfig))

	// 批量读取所有寄存器 (从第一个点表的地址开始)
	if len(pointConfig) == 0 {
		return points
	}

	// 计算需要读取的寄存器总数和起始地址
	startAddr := pointConfig[0].Address
	maxEndAddr := uint16(0)
	for _, p := range pointConfig {
		if p.Address < startAddr {
			startAddr = p.Address
		}
		endAddr := p.Address + p.Length
		if endAddr > maxEndAddr {
			maxEndAddr = endAddr
		}
	}
	totalLength := maxEndAddr - startAddr

	// 批量读取
	values, err := modbusrtu.ReadRegisters(serialTransceive, byte(devAddr), FUNC_CODE_READ, startAddr, totalLength, 1000, debug, 16, tinydrv.Logf)
	if err != nil {
		return points
	}

	// 将读取的值按点表配置转换为实际值
	for _, cfg := range pointConfig {
		offset := cfg.Address - startAddr
		if offset < 0 || int(offset) >= len(values) {
			continue
		}

		rawVal := values[offset]
		realVal := float64(rawVal) * cfg.Scale

		points = append(points, DriverPoint{
			FieldName: cfg.Field,
			Value:     tinydrv.FormatFloat(realVal, cfg.Decimals),
			RW:        cfg.RW,
			Unit:      cfg.Unit,
			Label:     cfg.Label,
		})
	}

	return points
}

// =============================================================================
// 【固定不变】Modbus RTU 通信函数
// =============================================================================

// 串口发送接收 (通用)
func serialTransceive(req []byte, respLen int, timeoutMs int) ([]byte, int) {
	return hostio.TransceiveBytes(callSerialTransceive, req, respLen, timeoutMs)
}

// 构建 Modbus RTU 读请求帧 (通用)
// =============================================================================
// 【固定不变】工具函数
// =============================================================================

// 获取配置 (通用)
func getConfig() DriverConfig {
	def := DriverConfig{DeviceAddress: 1, FuncName: "read"}
	config := tinydrv.ParseConfigMap()

	return DriverConfig{
		DeviceAddress: tinydrv.ParseInt(config, "device_address", def.DeviceAddress),
		FuncName:      tinydrv.ParseString(config, "func_name", def.FuncName),
		FieldName:     tinydrv.ParseString(config, "field_name", ""),
		Value:         tinydrv.ParseString(config, "value", ""),
		Debug:         tinydrv.ParseBool(config, "debug", false),
	}
}

// 格式化浮点数 (通用)
func main() {}
