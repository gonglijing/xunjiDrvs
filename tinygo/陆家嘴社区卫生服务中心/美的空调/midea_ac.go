// =============================================================================
// 美的空调 - Modbus RTU 驱动（陆家嘴社区卫生服务中心）
// =============================================================================
//
// 设备点表:
//   - 温度设点(TEMSET): FC=03, 地址=0, 长度=1, 表达式=v/10
//   - 湿度设点(HUMSET): FC=03, 地址=2, 长度=1, 表达式=v/10
//   - 环境温度(TEM): FC=03, 地址=48, 长度=1, 表达式=v/10
//   - 环境湿度(HUM): FC=03, 地址=49, 长度=1, 表达式=v/10
//   - 室内高温报警值(IHTAV): FC=03, 地址=17, 长度=1, 表达式=v/10
//   - 室内低温报警值(ILTAV): FC=03, 地址=18, 长度=1, 表达式=v/10
//   - 高湿度报警值(HHAV): FC=03, 地址=19, 长度=1, 表达式=v/10
//   - 低湿度报警值(LHAV): FC=03, 地址=20, 长度=1, 表达式=v/10
//   - 设备地址(ADD): FC=03, 地址=94, 长度=1, 表达式=v
//
// Host 提供: serial_transceive
//
// =============================================================================
package main

import (
	"math"
	"strconv"
	"strings"

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

// 宿主函数需要通过一层普通 Go 函数包装后，才能传给共享 helper。
func callSerialTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return serial_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

// =============================================================================
// 【固定不变】配置结构（网关传入）
// =============================================================================
type DriverConfig = tinydrv.DriverConfig

type DriverPoint = tinydrv.Point

type readPointSpec struct {
	Index    int
	Field    string
	Scale    float64
	Decimals int
	RW       string
	Unit     string
	Label    string
}

type readBlockSpec struct {
	Start  uint16
	Count  uint16
	Points []readPointSpec
}

type writablePointSpec struct {
	Field    string
	Register uint16
	Scale    float64
	Decimals int
	Unit     string
	Label    string
}

// =============================================================================
// 【用户修改】驱动版本
// =============================================================================
const (
	DriverVersion    = "1.0.0"
	DriverProductKey = "ljzchc_midea_ac"
)

// =============================================================================
// 【用户修改】点表定义
// =============================================================================
const (
	REG_TEMSET = 0
	REG_HUMSET = 2

	REG_IHTAV = 17
	REG_ILTAV = 18
	REG_HHAV  = 19
	REG_LHAV  = 20

	REG_TEM = 48
	REG_HUM = 49

	REG_ADD = 94

	FUNC_CODE_READ         = 0x03
	FUNC_CODE_WRITE_SINGLE = 0x06
)

var writablePointSpecs = []writablePointSpec{
	// 这些点本身就是设备参数项，现场通常会通过网关远程调整。
	{Field: "TEMSET", Register: REG_TEMSET, Scale: 0.1, Decimals: 1, Unit: "℃", Label: "温度设点"},
	{Field: "HUMSET", Register: REG_HUMSET, Scale: 0.1, Decimals: 1, Unit: "%", Label: "湿度设点"},
	{Field: "IHTAV", Register: REG_IHTAV, Scale: 0.1, Decimals: 1, Unit: "℃", Label: "室内高温报警值"},
	{Field: "ILTAV", Register: REG_ILTAV, Scale: 0.1, Decimals: 1, Unit: "℃", Label: "室内低温报警值"},
	{Field: "HHAV", Register: REG_HHAV, Scale: 0.1, Decimals: 1, Unit: "%", Label: "高湿度报警值"},
	{Field: "LHAV", Register: REG_LHAV, Scale: 0.1, Decimals: 1, Unit: "%", Label: "低湿度报警值"},
}

var readBlockSpecs = []readBlockSpec{
	{
		// 设定值段：
		// 0~2 这一小段同时包含温度设点和湿度设点，中间保留了一个未使用寄存器，
		// 因此点位索引分别落在 0 和 2。
		Start: REG_TEMSET,
		Count: 3,
		Points: []readPointSpec{
			{Index: 0, Field: "TEMSET", Scale: 0.1, Decimals: 1, RW: "RW", Unit: "℃", Label: "温度设点"},
			{Index: 2, Field: "HUMSET", Scale: 0.1, Decimals: 1, RW: "RW", Unit: "%", Label: "湿度设点"},
		},
	},
	{
		// 报警阈值段：
		// 17~20 连续保存温湿度上下限，属于参数类点位，因此统一标记为 RW。
		Start: REG_IHTAV,
		Count: 4,
		Points: []readPointSpec{
			{Index: 0, Field: "IHTAV", Scale: 0.1, Decimals: 1, RW: "RW", Unit: "℃", Label: "室内高温报警值"},
			{Index: 1, Field: "ILTAV", Scale: 0.1, Decimals: 1, RW: "RW", Unit: "℃", Label: "室内低温报警值"},
			{Index: 2, Field: "HHAV", Scale: 0.1, Decimals: 1, RW: "RW", Unit: "%", Label: "高湿度报警值"},
			{Index: 3, Field: "LHAV", Scale: 0.1, Decimals: 1, RW: "RW", Unit: "%", Label: "低湿度报警值"},
		},
	},
	{
		// 实时测量段：
		// 48~49 两个寄存器是现场当前环境量，只读。
		Start: REG_TEM,
		Count: 2,
		Points: []readPointSpec{
			{Index: 0, Field: "TEM", Scale: 0.1, Decimals: 1, RW: "R", Unit: "℃", Label: "环境温度"},
			{Index: 1, Field: "HUM", Scale: 0.1, Decimals: 1, RW: "R", Unit: "%", Label: "环境湿度"},
		},
	},
	{
		// 设备标识段：
		// 地址寄存器单独保留为一个块，使“辅助标识点”和业务点的区分仍然清晰。
		Start: REG_ADD,
		Count: 1,
		Points: []readPointSpec{
			{Index: 0, Field: "ADD", Scale: 1, Decimals: 1, RW: "R", Unit: "", Label: "设备地址"},
		},
	},
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
		point, err := writePoint(cfg.DeviceAddress, cfg.FieldName, cfg.Value, cfg.Debug)
		if err != nil {
			tinydrv.OutputHandleError(DriverProductKey, err.Error())
			return 0
		}
		tinydrv.OutputHandleSuccess(DriverProductKey, []DriverPoint{point})
		return 0
	}

	points := readAllPoints(cfg.DeviceAddress, cfg.Debug)
	tinydrv.OutputHandleSuccess(DriverProductKey, points)
	return 0
}

// =============================================================================
// 【固定不变】描述可写字段
// =============================================================================
//
//go:wasmexport describe
func describe() int32 {
	tinydrv.OutputDescribe(map[string]string{
		"language":        "tinygo",
		"transport":       "serial",
		"protocol":        "modbus_rtu",
		"write":           "single_point",
		"writable_fields": "TEMSET,HUMSET,IHTAV,ILTAV,HHAV,LHAV",
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
	points := make([]DriverPoint, 0, 9)
	addr := byte(devAddr)

	// 空调协议的寄存器天然分成几个离散区块。
	// 这里把“读哪一段”和“这一段里有哪些点”都收敛到表里，
	// 读取流程就退化成统一的“遍历区块 -> 按索引映射点位”。
	for _, block := range readBlockSpecs {
		values := readMultipleRegs(addr, block.Start, block.Count, debug)
		points = appendReadBlockPoints(points, values, block.Points)
	}

	return points
}

func appendReadBlockPoints(points []DriverPoint, values []int16, specs []readPointSpec) []DriverPoint {
	if values == nil {
		return points
	}

	for _, spec := range specs {
		if spec.Index < 0 || spec.Index >= len(values) {
			continue
		}
		points = append(points, makePoint(
			spec.Field,
			int(values[spec.Index]),
			spec.Scale,
			spec.Decimals,
			spec.RW,
			spec.Unit,
			spec.Label,
		))
	}

	return points
}

func writePoint(devAddr int, fieldName string, value string, debug bool) (DriverPoint, error) {
	spec, ok := findWritablePoint(fieldName)
	if !ok {
		return DriverPoint{}, modbusrtuErr("unsupported writable field")
	}

	rawValue, err := encodeWritableValue(spec, value)
	if err != nil {
		return DriverPoint{}, err
	}

	writtenReg, writtenValue, err := modbusrtu.WriteSingleRegister(
		serialTransceive,
		byte(devAddr),
		FUNC_CODE_WRITE_SINGLE,
		spec.Register,
		rawValue,
		1000,
		debug,
		24,
		tinydrv.Logf,
	)
	if err != nil {
		return DriverPoint{}, err
	}
	if writtenReg != spec.Register {
		return DriverPoint{}, modbusrtuErr("write register mismatch")
	}
	if writtenValue != rawValue {
		return DriverPoint{}, modbusrtuErr("write value mismatch")
	}

	return makePoint(spec.Field, int(writtenValue), spec.Scale, spec.Decimals, "RW", spec.Unit, spec.Label), nil
}

func makePoint(field string, rawVal int, scale float64, decimals int, rw, unit, label string) DriverPoint {
	// 这个辅助函数把“缩放 + 格式化 + 造点”固定下来，
	// 让调用处只保留设备点位本身的语义。
	realVal := float64(rawVal) * scale
	return makePointValue(field, realVal, decimals, rw, unit, label)
}

func makePointValue(field string, value float64, decimals int, rw, unit, label string) DriverPoint {
	return DriverPoint{
		FieldName: field,
		Value:     tinydrv.FormatFloat(value, decimals),
		RW:        rw,
		Unit:      unit,
		Label:     label,
	}
}

func findWritablePoint(fieldName string) (writablePointSpec, bool) {
	for _, spec := range writablePointSpecs {
		if strings.EqualFold(spec.Field, strings.TrimSpace(fieldName)) {
			return spec, true
		}
	}
	return writablePointSpec{}, false
}

func encodeWritableValue(spec writablePointSpec, value string) (uint16, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, err
	}
	raw := math.Round(parsed / spec.Scale)
	if raw < 0 || raw > 65535 {
		return 0, modbusrtuErr("write value out of range")
	}
	return uint16(raw), nil
}

type modbusrtuErr string

func (e modbusrtuErr) Error() string { return string(e) }

// =============================================================================
// 【固定不变】Modbus RTU 通信函数
// =============================================================================

func readMultipleRegs(devAddr byte, startReg uint16, count uint16, debug bool) []int16 {
	// 通信 helper 返回的是 []uint16，这里再转成 []int16，
	// 便于设备层在需要时按有符号值继续处理。
	values, err := modbusrtu.ReadRegisters(serialTransceive, devAddr, FUNC_CODE_READ, startReg, count, 1000, debug, 24, tinydrv.Logf)
	if err != nil {
		return nil
	}

	result := make([]int16, count)
	for i := 0; i < int(count); i++ {
		result[i] = int16(values[i])
	}
	return result
}

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
