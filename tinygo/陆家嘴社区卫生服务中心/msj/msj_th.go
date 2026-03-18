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
	"strconv"

	pdk "github.com/extism/go-pdk"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/modbusrtu"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/tinydrv"
)

//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

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

	logf("readAllPoints: devAddr=%d, debug=%v", devAddr, debug)

	values := readMultipleRegs(byte(devAddr), REG_TEMPERATURE, uint16(len(pointDefs)), debug)
	logf("readMultipleRegs returned: %v (len=%d)", values, len(values))

	if values == nil || len(values) < len(pointDefs) {
		logf("not enough values, returning empty")
		return points
	}

	for index, def := range pointDefs {
		points = append(points, makePoint(def, int64(values[index])))
	}

	logf("returning %d points", len(points))
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
	req := buildReadFrame(devAddr, startReg, count)
	if debug {
		logf("rtu req=% X", req)
	}

	resp, n := serialTransceive(req, int(count)*2+5, 1000)
	if debug {
		logf("rtu n=%d resp=%s", n, hexPreview(resp, n, 16))
	}
	if n <= 0 {
		return nil
	}

	values, err := parseReadResponse(resp[:n], devAddr)
	if err != nil {
		if debug {
			logf("parse err=%v", err)
		}
		return nil
	}
	return values
}

func serialTransceive(req []byte, respLen int, timeoutMs int) ([]byte, int) {
	if len(req) == 0 || respLen <= 0 {
		return nil, 0
	}

	reqMem := pdk.AllocateBytes(req)
	defer reqMem.Free()
	respMem := pdk.Allocate(respLen)
	defer respMem.Free()

	n := int(serial_transceive(
		reqMem.Offset(), uint64(len(req)),
		respMem.Offset(), uint64(respLen),
		uint64(timeoutMs),
	))
	if n <= 0 {
		return nil, n
	}
	if n > respLen {
		n = respLen
	}

	resp := make([]byte, n)
	mem := pdk.NewMemory(respMem.Offset(), uint64(n))
	mem.Load(resp)
	return resp, n
}

func buildReadFrame(addr byte, start uint16, qty uint16) []byte {
	return modbusrtu.BuildReadFrame(addr, FUNC_CODE_READ_HOLDING, start, qty)
}

func parseReadResponse(data []byte, addr byte) ([]uint16, error) {
	return modbusrtu.ParseReadResponse(data, addr, FUNC_CODE_READ_HOLDING)
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

func formatFloat(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

func outputJSON(v interface{}) {
	tinydrv.OutputJSON(v)
}

func logf(format string, args ...interface{}) {
	tinydrv.Logf(format, args...)
}

func hexPreview(b []byte, n int, max int) string {
	return tinydrv.HexPreview(b, n, max)
}

func main() {}
