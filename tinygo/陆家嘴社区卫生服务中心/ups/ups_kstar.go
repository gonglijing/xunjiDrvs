// =============================================================================
// 科士达 UPS - Modbus TCP 驱动（陆家嘴卫生活动中心）
// =============================================================================
//
// 设备点表:
//   - R相输出电压(OUR): FC=03, 地址=120, 长度=1, 缩放=0.1
//   - S相输出电压(OUS): FC=03, 地址=121, 长度=1, 缩放=0.1
//   - T相输出电压(OUT): FC=03, 地址=122, 长度=1, 缩放=0.1
//   - 输出频率(OH): FC=03, 地址=119, 长度=1, 缩放=0.1
//   - R相负载率(loadR): FC=03, 地址=123, 长度=1, 缩放=1
//   - S相负载率(loadS): FC=03, 地址=124, 长度=1, 缩放=1
//   - T相负载率(loadT): FC=03, 地址=125, 长度=1, 缩放=1
//   - R相输入电压(IUR): FC=03, 地址=110, 长度=1, 缩放=0.1
//   - S相输入电压(IUS): FC=03, 地址=111, 长度=1, 缩放=0.1
//   - T相输入电压(IOT): FC=03, 地址=112, 长度=1, 缩放=0.1
//   - 输入频率(IH): FC=03, 地址=109, 长度=1, 缩放=0.1
//   - 电池容量(qos): FC=03, 地址=100, 长度=1, 缩放=0.1
//   - 电池剩余时间(ltime): FC=03, 地址=101, 长度=1, 缩放=1
//
// Host 提供: tcp_transceive
//
// =============================================================================
package main

import (
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/hostio"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/modbustcp"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/tinydrv"
)

// =============================================================================
// 【固定不变】Host 函数声明
// =============================================================================
//
//go:wasmimport extism:host/user tcp_transceive
func tcp_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

// 本地包装层用于把宿主 TCP 调用传给共享 helper。
func callTCPTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return tcp_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

// =============================================================================
// 【固定不变】配置结构（网关传入）
// =============================================================================
type DriverConfig = tinydrv.DriverConfig

type DriverPoint = tinydrv.Point

type blockPointSpec struct {
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
	Points []blockPointSpec
}

// =============================================================================
// 【用户修改】驱动版本
// =============================================================================
const (
	DriverVersion    = "1.0.0"
	DriverProductKey = "ljzchc_ups_kstar"
)

// =============================================================================
// 【用户修改】点表定义
// =============================================================================
const (
	REG_BATTERY_CAPACITY    = 100 // 电池容量 qos
	REG_BATTERY_REMAIN_TIME = 101 // 电池剩余时间 ltime

	REG_INPUT_FREQUENCY = 109 // 输入频率 IH
	REG_INPUT_VOLTAGE_R = 110 // R相输入电压 IUR
	REG_INPUT_VOLTAGE_S = 111 // S相输入电压 IUS
	REG_INPUT_VOLTAGE_T = 112 // T相输入电压 IOT

	REG_OUTPUT_FREQUENCY = 119 // 输出频率 OH
	REG_OUTPUT_VOLTAGE_R = 120 // R相输出电压 OUR
	REG_OUTPUT_VOLTAGE_S = 121 // S相输出电压 OUS
	REG_OUTPUT_VOLTAGE_T = 122 // T相输出电压 OUT
	REG_LOAD_PERCENT_R   = 123 // R相负载率 loadR
	REG_LOAD_PERCENT_S   = 124 // S相负载率 loadS
	REG_LOAD_PERCENT_T   = 125 // T相负载率 loadT

	FUNC_CODE_READ = 0x03 // 读保持寄存器
)

var readBlockSpecs = []readBlockSpec{
	{
		// 输出段：
		// 这一段同时覆盖输出频率、三相输出电压和三相负载率。
		Start: REG_OUTPUT_FREQUENCY,
		Count: 7,
		Points: []blockPointSpec{
			{Index: 1, Field: "OUR", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "R相输出电压"},
			{Index: 2, Field: "OUS", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "S相输出电压"},
			{Index: 3, Field: "OUT", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "T相输出电压"},
			{Index: 0, Field: "OH", Scale: 0.1, Decimals: 1, RW: "R", Unit: "Hz", Label: "输出频率"},
			{Index: 4, Field: "loadR", Scale: 1, Decimals: 0, RW: "R", Unit: "%", Label: "R相负载率"},
			{Index: 5, Field: "loadS", Scale: 1, Decimals: 0, RW: "R", Unit: "%", Label: "S相负载率"},
			{Index: 6, Field: "loadT", Scale: 1, Decimals: 0, RW: "R", Unit: "%", Label: "T相负载率"},
		},
	},
	{
		// 输入段：
		// 输入频率和三相输入电压都在这 4 个连续寄存器里。
		Start: REG_INPUT_FREQUENCY,
		Count: 4,
		Points: []blockPointSpec{
			{Index: 1, Field: "IUR", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "R相输入电压"},
			{Index: 2, Field: "IUS", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "S相输入电压"},
			{Index: 3, Field: "IOT", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "T相输入电压"},
			{Index: 0, Field: "IH", Scale: 0.1, Decimals: 1, RW: "R", Unit: "Hz", Label: "输入频率"},
		},
	},
	{
		// 电池段：
		// 电池容量和剩余时间天然是一组连续状态量。
		Start: REG_BATTERY_CAPACITY,
		Count: 2,
		Points: []blockPointSpec{
			{Index: 0, Field: "qos", Scale: 0.1, Decimals: 1, RW: "R", Unit: "%", Label: "电池容量"},
			{Index: 1, Field: "ltime", Scale: 1, Decimals: 0, RW: "R", Unit: "min", Label: "电池剩余时间"},
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
		return writeNotSupported(cfg.FieldName)
	}
	points := readAllPoints(cfg.DeviceAddress)

	tinydrv.OutputHandleSuccess(DriverProductKey, points)
	return 0
}

func writeNotSupported(fieldName string) int32 {
	errText := "UPS 驱动当前仅支持状态采集，不支持写入"
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
		"transport": "tcp",
		"protocol":  "modbus_tcp",
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
		"transport": "tcp",
		"protocol":  "modbus_tcp",
	})
	return 0
}

// =============================================================================
// 【用户修改】读取所有测点
// =============================================================================
func readAllPoints(devAddr int) []DriverPoint {
	points := make([]DriverPoint, 0, 13)
	addr := byte(devAddr)

	// UPS 点位天然分成输出侧、输入侧、电池侧三个区块。
	// 这里把块定义和块内点位映射都写到表里，读取流程就只剩一个清晰的遍历。
	for _, block := range readBlockSpecs {
		values := readMultipleRegs(addr, block.Start, block.Count)
		points = appendBlockPoints(points, values, block.Points)
	}

	return points
}

func appendBlockPoints(points []DriverPoint, values []int16, specs []blockPointSpec) []DriverPoint {
	if values == nil {
		return points
	}

	for _, spec := range specs {
		if spec.Index < 0 || spec.Index >= len(values) {
			continue
		}
		points = append(points, makePoint(spec, values[spec.Index]))
	}

	return points
}

func makePoint(spec blockPointSpec, rawVal int16) DriverPoint {
	// 统一处理缩放和格式化，让调用处保留纯粹的点位定义。
	realVal := float64(rawVal) * spec.Scale
	return DriverPoint{
		FieldName: spec.Field,
		Value:     tinydrv.FormatFloat(realVal, spec.Decimals),
		RW:        spec.RW,
		Unit:      spec.Unit,
		Label:     spec.Label,
	}
}

// =============================================================================
// 【固定不变】Modbus TCP 通信函数
// =============================================================================

func readMultipleRegs(devAddr byte, startReg uint16, count uint16) []int16 {
	// 公共 TCP helper 返回 []uint16，这里再转成 int16，
	// 保持驱动层看到的仍是“一个寄存器一个数值”的简单模型。
	values, err := modbustcp.ReadRegisters(tcpTransceive, devAddr, FUNC_CODE_READ, startReg, count, 1000, 64)
	if err != nil {
		return nil
	}

	result := make([]int16, count)
	for i := 0; i < int(count); i++ {
		result[i] = int16(values[i])
	}
	return result
}

func tcpTransceive(req []byte, resp []byte, timeoutMs int) int {
	return hostio.TransceiveInto(callTCPTransceive, req, resp, timeoutMs)
}

// =============================================================================
// 【固定不变】工具函数
// =============================================================================

func getConfig() DriverConfig {
	return tinydrv.ParseDriverConfig(DriverConfig{DeviceAddress: 1, FuncName: "read"})
}

func main() {}
