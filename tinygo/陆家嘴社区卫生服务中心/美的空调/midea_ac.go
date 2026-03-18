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
	"strconv"

	pdk "github.com/extism/go-pdk"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/modbusrtu"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/tinydrv"
)

// =============================================================================
// 【固定不变】Host 函数声明
// =============================================================================
//
//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

// =============================================================================
// 【固定不变】配置结构（网关传入）
// =============================================================================
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

	FUNC_CODE_READ = 0x03
)

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
	points := make([]DriverPoint, 0, 9)

	if values := readMultipleRegs(byte(devAddr), REG_TEMSET, 3, debug); values != nil {
		points = append(points, makePoint("TEMSET", int(values[0]), 0.1, 1, "R", "℃", "温度设点"))
		points = append(points, makePoint("HUMSET", int(values[2]), 0.1, 1, "R", "%", "湿度设点"))
	}

	if values := readMultipleRegs(byte(devAddr), REG_IHTAV, 4, debug); values != nil {
		points = append(points, makePoint("IHTAV", int(values[0]), 0.1, 1, "R", "℃", "室内高温报警值"))
		points = append(points, makePoint("ILTAV", int(values[1]), 0.1, 1, "R", "℃", "室内低温报警值"))
		points = append(points, makePoint("HHAV", int(values[2]), 0.1, 1, "R", "%", "高湿度报警值"))
		points = append(points, makePoint("LHAV", int(values[3]), 0.1, 1, "R", "%", "低湿度报警值"))
	}

	if values := readMultipleRegs(byte(devAddr), REG_TEM, 2, debug); values != nil {
		points = append(points, makePoint("TEM", int(values[0]), 0.1, 1, "R", "℃", "环境温度"))
		points = append(points, makePoint("HUM", int(values[1]), 0.1, 1, "R", "%", "环境湿度"))
	}

	if val := readSingleReg(byte(devAddr), REG_ADD, debug); val >= 0 {
		points = append(points, makePoint("ADD", int(val), 1, 1, "R", "", "设备地址"))
	}

	return points
}

func makePoint(field string, rawVal int, scale float64, decimals int, rw, unit, label string) DriverPoint {
	realVal := float64(rawVal) * scale
	return makePointValue(field, realVal, decimals, rw, unit, label)
}

func makePointValue(field string, value float64, decimals int, rw, unit, label string) DriverPoint {
	return DriverPoint{
		FieldName: field,
		Value:     formatFloat(value, decimals),
		RW:        rw,
		Unit:      unit,
		Label:     label,
	}
}

// =============================================================================
// 【固定不变】Modbus RTU 通信函数
// =============================================================================

func readSingleReg(devAddr byte, regAddr uint16, debug bool) int {
	values := readMultipleRegs(devAddr, regAddr, 1, debug)
	if values == nil || len(values) < 1 {
		return -1
	}
	return int(values[0])
}

func readMultipleRegs(devAddr byte, startReg uint16, count uint16, debug bool) []int16 {
	req := buildReadFrame(devAddr, startReg, count)
	if debug {
		logf("rtu req=% X", req)
	}

	resp, n := serialTransceive(req, int(count)*2+5, 1000)
	if debug {
		logf("rtu n=%d resp=%s", n, hexPreview(resp, n, 24))
	}
	if n <= 0 {
		return nil
	}

	values, err := parseReadResponse(resp[:n], devAddr)
	if err != nil || len(values) < int(count) {
		if debug {
			logf("parse err=%v", err)
		}
		return nil
	}

	result := make([]int16, count)
	for i := 0; i < int(count); i++ {
		result[i] = int16(values[i])
	}
	return result
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
	return modbusrtu.BuildReadFrame(addr, FUNC_CODE_READ, start, qty)
}

func parseReadResponse(data []byte, addr byte) ([]uint16, error) {
	return modbusrtu.ParseReadResponse(data, addr, FUNC_CODE_READ)
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

func hexPreview(b []byte, n int, max int) string {
	return tinydrv.HexPreview(b, n, max)
}

func main() {}
