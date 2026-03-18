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

// 本地包装层用于把宿主调用传入 hostio/modbusrtu 等共享 helper。
func callSerialTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return serial_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

// =============================================================================
// 【固定不变】配置结构（网关传入）
// =============================================================================
type DriverConfig = tinydrv.DriverConfig

type DriverPoint = tinydrv.Point
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
	// 液位段：
	// 液位值占用 2 个寄存器，是这个设备最核心的业务点位。
	// 后续计算会把原始压力/高度相关值转换成现场需要展示的液位工程量。
	{Field: "level", Address: REG_LEVEL, Length: 2, Decimals: 3, RW: "R", Unit: "", Label: "液位"},

	// 温度段：
	// 温度是附带环境量，只占 1 个寄存器，但仍保留在同一张点表里，便于统一维护。
	{Field: "wtemp", Address: REG_WTEMP, Length: 1, Decimals: 2, RW: "R", Unit: "°C", Label: "温度"},
}

// 点表配置结构
type PointConfig struct {
	// 液位传感器既有 2 寄存器的液位值，也有 1 寄存器的温度值，
	// 因此 Length 是点表里不可缺少的一列。
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
			tinydrv.OutputHandleError(DriverProductKey, "panic")
		}
	}()

	cfg := getConfig()
	if tinydrv.IsWriteFunc(cfg.FuncName) {
		return writeNotSupported(cfg.FieldName)
	}

	points := readAllPoints(cfg.DeviceAddress, cfg.Debug)

	tinydrv.OutputHandleSuccess(DriverProductKey, points)
	return 0
}

func writeNotSupported(fieldName string) int32 {
	errText := "液位传感器驱动当前仅支持读取，不支持写入"
	if fieldName != "" {
		errText += ": " + fieldName
	}
	tinydrv.OutputHandleError(DriverProductKey, errText)
	return 0
}

// =============================================================================
// 【固定不变】描述可写字段
// =============================================================================
//
//go:wasmexport describe
func describe() int32 {
	tinydrv.OutputDescribe(map[string]string{
		"language":  "tinygo",
		"transport": "serial",
		"protocol":  "modbus_rtu",
		"write":     "unsupported",
	})
	return 0
}

// =============================================================================
// 【固定不变】驱动版本
// =============================================================================
//
//go:wasmexport version
func version() int32 {
	tinydrv.OutputVersion(DriverVersion, DriverProductKey, map[string]string{
		"language":  "tinygo",
		"transport": "serial",
		"protocol":  "modbus_rtu",
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

	// 和压力传感器一样，这里根据点表自动推导出连续读取区间，
	// 减少“点表”和“通信区间”两份配置之间漂移的风险。
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

	values, err := modbusrtu.ReadRegisters(serialTransceive, byte(devAddr), FUNC_CODE_READ, startAddr, totalLength, 1000, debug, 16, tinydrv.Logf)
	if err != nil {
		return points
	}

	// 先按寄存器长度还原原始整数，再套用字段对应的工程量公式。
	for _, cfg := range pointConfig {
		offset := int(cfg.Address - startAddr)
		if offset < 0 || offset+int(cfg.Length) > len(values) {
			continue
		}

		rawVal := combineRegisters(values[offset : offset+int(cfg.Length)])
		realVal := applyExpression(cfg.Field, rawVal)

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

func combineRegisters(words []uint16) int64 {
	// 多寄存器值按高位在前拼接；当前设备只用到 1 或 2 个寄存器，
	// 但保留更一般化实现后，后续扩点时不用再改这里。
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
	// 协议里的换算公式显式写在这里，比把魔法数字分散在点表或调用点里更好读。
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
	return tinydrv.ParseDriverConfig(DriverConfig{DeviceAddress: 1, FuncName: "read"})
}

func main() {}
