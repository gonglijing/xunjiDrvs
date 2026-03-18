// =============================================================================
// 共济温湿度 - Modbus RTU 驱动（陆家嘴社区卫生服务中心）
// =============================================================================
//
// 协议类型: Modbus RTU
// 功能码: 0x04 (INPUT_REGISTER)
//
// 点表:
//   - 温度(temperature): 地址=0, 长度=1, 表达式=v/10
//   - 湿度(humidity): 地址=1, 长度=1, 表达式=v/10
//   - 漏点温度(dewtemperature): 地址=2, 长度=1, 表达式=v/10
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

//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

// TinyGo 当前不能把 wasmimport 函数直接当作函数值传给共享 helper，
// 因此这里保留一个极薄的本地包装层。
func callSerialTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return serial_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

// DriverConfig 描述网关注入给驱动的运行参数。
// 这个驱动真正关心的是 device_address 与 debug，其余字段保留是为了与统一接口对齐。
type DriverConfig = tinydrv.DriverConfig

type DriverPoint = tinydrv.Point

// pointDef 只描述“一个业务点如何从寄存器值换算出来”，
// 不掺杂任何通信层细节。
type pointDef struct {
	Field    string
	Scale    float64
	Decimals int
	RW       string
	Unit     string
	Label    string
}

const (
	DriverVersion    = "1.0.0"
	DriverProductKey = "ljzchc_gongji_th"
)

const (
	REG_TEMPERATURE     = 0
	REG_HUMIDITY        = 1
	REG_DEW_TEMPERATURE = 2

	FUNC_CODE_READ_INPUT = 0x04
)

var pointDefs = [...]pointDef{
	{Field: "temperature", Scale: 0.1, Decimals: 1, RW: "R", Unit: "℃", Label: "温度"},
	{Field: "humidity", Scale: 0.1, Decimals: 1, RW: "R", Unit: "%", Label: "湿度"},
	{Field: "dewtemperature", Scale: 0.1, Decimals: 1, RW: "R", Unit: "℃", Label: "漏点温度"},
}

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

//go:wasmexport version
func version() int32 {
	tinydrv.OutputVersion(DriverVersion, DriverProductKey, map[string]string{
		"language":  "tinygo",
		"transport": "serial",
		"protocol":  "modbus_rtu",
	})
	return 0
}

func readAllPoints(devAddr int, debug bool) []DriverPoint {
	points := make([]DriverPoint, 0, len(pointDefs))

	// 这台设备的点位正好落在一段连续寄存器里，因此一次读完最容易理解，也最省通信次数。
	values := readMultipleRegs(byte(devAddr), REG_TEMPERATURE, uint16(len(pointDefs)), debug)
	if values == nil || len(values) < len(pointDefs) {
		return points
	}

	for index, def := range pointDefs {
		points = append(points, makePoint(def, int64(values[index])))
	}

	return points
}

func writeNotSupported(fieldName string) int32 {
	errText := "共济温湿度驱动当前仅支持读取，不支持写入"
	if fieldName != "" {
		errText += ": " + fieldName
	}
	tinydrv.OutputHandleError(DriverProductKey, errText)
	return 0
}

func makePoint(def pointDef, raw int64) DriverPoint {
	// 共济温湿度的三个点都遵循“寄存器原值 * 缩放系数”的简单模式。
	v := float64(raw) * def.Scale
	return DriverPoint{
		FieldName: def.Field,
		Value:     tinydrv.FormatFloat(v, def.Decimals),
		RW:        def.RW,
		Unit:      def.Unit,
		Label:     def.Label,
	}
}

func readMultipleRegs(devAddr byte, startReg uint16, count uint16, debug bool) []uint16 {
	// 通信样板已经沉到 modbusrtu 包，这里只保留设备自己的功能码与调试策略。
	values, err := modbusrtu.ReadRegisters(serialTransceive, devAddr, FUNC_CODE_READ_INPUT, startReg, count, 1000, debug, 16, tinydrv.Logf)
	if err != nil {
		return nil
	}
	return values
}

func serialTransceive(req []byte, respLen int, timeoutMs int) ([]byte, int) {
	return hostio.TransceiveBytes(callSerialTransceive, req, respLen, timeoutMs)
}

func getConfig() DriverConfig {
	// 配置解析统一走 tinydrv，保证默认值和 trim 行为与其他驱动一致。
	return tinydrv.ParseDriverConfig(DriverConfig{DeviceAddress: 1, FuncName: "read"})
}

func main() {}
