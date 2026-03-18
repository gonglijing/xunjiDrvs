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
			tinydrv.OutputJSON(ErrorResponse{Success: false, Error: "panic"})
		}
	}()

	cfg := getConfig()
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
		Value:     tinydrv.FormatFloat(value, decimals),
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

func main() {}
