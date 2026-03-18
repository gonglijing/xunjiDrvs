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
	"strconv"

	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/hostio"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/modbusrtu"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/tinydrv"
)

//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

func callSerialTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return serial_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

type DriverConfig struct {
	DeviceAddress int    `json:"device_address"`
	FuncName      string `json:"func_name"`
	FieldName     string `json:"field_name"`
	Value         string `json:"value"`
	Debug         bool   `json:"debug"`
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

//go:wasmexport describe
func describe() int32 {
	outputJSON(DescribeResponse{Success: true})
	return 0
}

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

func readAllPoints(devAddr int, debug bool) []DriverPoint {
	points := make([]DriverPoint, 0, len(pointDefs))

	values := readMultipleRegs(byte(devAddr), REG_TEMPERATURE, uint16(len(pointDefs)), debug)
	if values == nil || len(values) < len(pointDefs) {
		return points
	}

	for index, def := range pointDefs {
		points = append(points, makePoint(def, int64(values[index])))
	}

	return points
}

func makePoint(def pointDef, raw int64) DriverPoint {
	v := float64(raw) * def.Scale
	return DriverPoint{
		FieldName: def.Field,
		Value:     formatFloat(v, def.Decimals),
		RW:        def.RW,
		Unit:      def.Unit,
		Label:     def.Label,
	}
}

func readMultipleRegs(devAddr byte, startReg uint16, count uint16, debug bool) []uint16 {
	values, err := modbusrtu.ReadRegisters(serialTransceive, devAddr, FUNC_CODE_READ_INPUT, startReg, count, 1000, debug, 16, logf)
	if err != nil {
		return nil
	}
	return values
}

func serialTransceive(req []byte, respLen int, timeoutMs int) ([]byte, int) {
	return hostio.TransceiveBytes(callSerialTransceive, req, respLen, timeoutMs)
}

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
