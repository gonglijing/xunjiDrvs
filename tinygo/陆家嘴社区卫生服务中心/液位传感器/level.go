// =============================================================================
// 液位传感器 - Modbus RTU 驱动
// =============================================================================
//
// 设备点表:
//   - 液位(level): FC=03(HOLDING_REGISTER), 地址=0x0000, 长度=2
//     数据类型=int64, 读写=R, 表达式=(v-101665)/9800, 小数位=3
//   - 温度(wtemp): FC=03(HOLDING_REGISTER), 地址=0x0002, 长度=1
//     数据类型=int64, 读写=R, 表达式=v/100, 小数位=2
//
// Host 提供: serial_transceive
//
// =============================================================================
package main

import (
	"strconv"

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

type HandleResponse struct {
	Success    bool          `json:"success"`
	ProductKey string        `json:"productKey"`
	Points     []DriverPoint `json:"points"`
	Error      string        `json:"error,omitempty"`
}

type DescribeResponse = tinydrv.DescribeResponse

type VersionData = tinydrv.VersionData

type VersionResponse = tinydrv.VersionResponse

type ErrorResponse = tinydrv.ErrorResponse

// =============================================================================
// 【用户修改】驱动版本
// =============================================================================
const (
	DriverVersion    = "1.0.0"
	DriverProductKey = "ljzchc_level_sensor"
)

// =============================================================================
// 【用户修改】点表定义
// =============================================================================
const (
	// 寄存器地址定义
	// 格式: RegisterName = Address // 说明
	REG_LEVEL = 0x0000 // 液位寄存器(2个寄存器)
	REG_WTEMP = 0x0002 // 温度寄存器

	// 功能码定义
	FUNC_CODE_READ = 0x03 // 读保持寄存器
)

// =============================================================================
// 【用户修改】点表配置
// =============================================================================
var pointConfig = []PointConfig{
	{Field: "level", Address: REG_LEVEL, Length: 2, Decimals: 3, RW: "R", Unit: "", Label: "液位"},
	{Field: "wtemp", Address: REG_WTEMP, Length: 1, Decimals: 2, RW: "R", Unit: "°C", Label: "温度"},
}

// 点表配置结构
type PointConfig struct {
	Field    string // 字段名
	Address  uint16 // 寄存器地址
	Length   uint16 // 寄存器数量
	Decimals int    // 有效小数位数
	RW       string // 读写属性
	Unit     string // 单位
	Label    string // 显示标签
}

// =============================================================================
// 【固定不变】驱动入口
// =============================================================================
//
//go:wasmexport handle
func handle() int32 {
	defer func() {
		if r := recover(); r != nil {
			outputJSON(ErrorResponse{Success: false, Error: "panic"})
		}
	}()

	cfg := getConfig()

	points := readAllPoints(cfg.DeviceAddress, cfg.Debug)

	outputJSON(HandleResponse{
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
	outputJSON(DescribeResponse{Success: true})
	return 0
}

// =============================================================================
// 【固定不变】驱动版本
// =============================================================================
//
//go:wasmexport version
func version() int32 {
	outputJSON(VersionResponse{
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
func readAllPoints(devAddr int, debug bool) []DriverPoint {
	points := make([]DriverPoint, 0, len(pointConfig))

	if len(pointConfig) == 0 {
		return points
	}

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

	values, err := modbusrtu.ReadRegisters(serialTransceive, byte(devAddr), FUNC_CODE_READ, startAddr, totalLength, 1000, debug, 16, logf)
	if err != nil {
		return points
	}

	for _, cfg := range pointConfig {
		offset := int(cfg.Address - startAddr)
		if offset < 0 || offset+int(cfg.Length) > len(values) {
			continue
		}

		rawVal := combineRegisters(values[offset : offset+int(cfg.Length)])
		realVal := applyExpression(cfg.Field, rawVal)

		points = append(points, DriverPoint{
			FieldName: cfg.Field,
			Value:     formatFloat(realVal, cfg.Decimals),
			RW:        cfg.RW,
			Unit:      cfg.Unit,
			Label:     cfg.Label,
		})
	}

	return points
}

func combineRegisters(words []uint16) int64 {
	if len(words) == 0 {
		return 0
	}
	if len(words) == 1 {
		return int64(words[0])
	}

	var v uint32
	for _, w := range words {
		v = (v << 16) | uint32(w)
	}
	return int64(v)
}

func applyExpression(field string, raw int64) float64 {
	v := float64(raw)
	switch field {
	case "level":
		return (v - 101665.0) / 9800.0
	case "wtemp":
		return v / 100.0
	default:
		return v
	}
}

// =============================================================================
// 【固定不变】Modbus RTU 通信函数
// =============================================================================

func serialTransceive(req []byte, respLen int, timeoutMs int) ([]byte, int) {
	return hostio.TransceiveBytes(callSerialTransceive, req, respLen, timeoutMs)
}

// =============================================================================
// 【固定不变】工具函数
// =============================================================================

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

func formatFloat(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

func outputJSON(v interface{}) {
	tinydrv.OutputJSON(v)
}

func logf(format string, args ...interface{}) {
	tinydrv.Logf(format, args...)
}

func main() {}
