// =============================================================================
// 共济温湿度 - Modbus RTU 驱动（陆家嘴社区卫生服务中心）
// =============================================================================
//
// 协议类型: Modbus RTU
// 功能码: 0x03 (INPUT_REGISTER)
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

// 与其他 TinyGo 驱动一致，这里通过本地包装层把宿主调用传给共享 helper。
func callSerialTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return serial_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

// msj 驱动沿用统一配置结构，其中 debug 默认开启，方便现场联调时直接看到请求/响应。
type DriverConfig struct {
	DeviceAddress int    `json:"device_address"`
	FuncName      string `json:"func_name"`
	FieldName     string `json:"field_name"`
	Value         string `json:"value"`
	Debug         bool   `json:"debug"`
}

type DriverPoint = tinydrv.Point
type HandleResponse = tinydrv.HandleResponse

type DescribeResponse = tinydrv.DescribeResponse
type VersionData = tinydrv.VersionData
type VersionResponse = tinydrv.VersionResponse
type ErrorResponse = tinydrv.ErrorResponse

// 这个驱动的点位模型与共济温湿度几乎一致，都是少量连续寄存器 + 简单缩放。
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
	DriverProductKey = "ljzchc_msj_th"
)

const (
	REG_TEMPERATURE = 0
	REG_HUMIDITY    = 1

	FUNC_CODE_READ_HOLDING = 0x03 // 读保持寄存器
)

var pointDefs = [...]pointDef{
	{Field: "temperature", Scale: 0.1, Decimals: 1, RW: "R", Unit: "℃", Label: "温度"},
	{Field: "humidity", Scale: 0.1, Decimals: 1, RW: "R", Unit: "%", Label: "湿度"},
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
	tinydrv.OutputJSON(DescribeResponse{Success: true})
	return 0
}

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

func readAllPoints(devAddr int, debug bool) []DriverPoint {
	points := make([]DriverPoint, 0, len(pointDefs))

	// 设备点位少而连续，因此保持最直接的“读一段 -> 按表造点”流程。
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
	errText := "msj 温湿度驱动当前仅支持读取，不支持写入"
	if fieldName != "" {
		errText += ": " + fieldName
	}
	tinydrv.OutputHandleError(DriverProductKey, errText)
	return 0
}

func makePoint(def pointDef, raw int64) DriverPoint {
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
	// 与共济温湿度不同，这台设备使用 0x03 读保持寄存器。
	values, err := modbusrtu.ReadRegisters(serialTransceive, devAddr, FUNC_CODE_READ_HOLDING, startReg, count, 1000, debug, 16, tinydrv.Logf)
	if err != nil {
		return nil
	}
	return values
}

func serialTransceive(req []byte, respLen int, timeoutMs int) ([]byte, int) {
	return hostio.TransceiveBytes(callSerialTransceive, req, respLen, timeoutMs)
}

func getConfig() DriverConfig {
	def := DriverConfig{DeviceAddress: 1, FuncName: "read", Debug: true} // 默认开启调试
	config := tinydrv.ParseConfigMap()

	return DriverConfig{
		DeviceAddress: tinydrv.ParseInt(config, "device_address", def.DeviceAddress),
		FuncName:      tinydrv.ParseString(config, "func_name", def.FuncName),
		FieldName:     tinydrv.ParseString(config, "field_name", ""),
		Value:         tinydrv.ParseString(config, "value", ""),
		Debug:         tinydrv.ParseBool(config, "debug", def.Debug),
	}
}

func main() {}
