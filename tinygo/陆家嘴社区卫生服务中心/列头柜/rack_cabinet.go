// =============================================================================
// 列头柜 - Modbus TCP 驱动（陆家嘴社区卫生服务中心）
// =============================================================================
//
// 协议类型: Modbus TCP
// 功能码: 0x03 (HOLDING_REGISTER)
//
// Host 提供: tcp_transceive
//
// =============================================================================
package main

import (
	"strconv"

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

// 把 wasmimport 包装成普通函数，便于传给共享 TCP helper。
func callTCPTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return tcp_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

// =============================================================================
// 【固定不变】配置结构（网关传入）
// =============================================================================
type DriverConfig struct {
	DeviceAddress int    `json:"device_address"`
	FuncName      string `json:"func_name"`
	FieldName     string `json:"field_name"`
	Value         string `json:"value"`
}

type DriverPoint = tinydrv.Point
type HandleResponse = tinydrv.HandleResponse

type DescribeResponse = tinydrv.DescribeResponse
type VersionData = tinydrv.VersionData
type VersionResponse = tinydrv.VersionResponse
type ErrorResponse = tinydrv.ErrorResponse

// blockPointSpec 用于描述“连续寄存器块里某个偏移位置对应一个普通数值点”。
type blockPointSpec struct {
	Index    int
	Field    string
	Scale    float64
	Decimals int
	RW       string
	Unit     string
	Label    string
}

type energyPointSpec struct {
	// 电能点位跨两个寄存器，且在块内的位置并不全是简单顺序，因此保留真实寄存器地址。
	Register uint16
	Field    string
	Scale    float64
	Decimals int
	RW       string
	Unit     string
	Label    string
}

type switchPointSpec struct {
	// 开关点的特殊之处在于最终关心的是某个状态位，而不是整个寄存器的工程量。
	Index int
	Field string
	Label string
}

// =============================================================================
// 【用户修改】驱动版本
// =============================================================================
const (
	DriverVersion    = "1.0.0"
	DriverProductKey = "ljzchc_rack_cabinet"
)

// =============================================================================
// 【用户修改】点表定义
// =============================================================================
const (
	FUNC_CODE_READ = 0x03

	REG_SWITCH_START  = 170
	REG_SWITCH_LEN    = 17
	REG_VOLTAGE_START = 275
	REG_VOLTAGE_LEN   = 4
	REG_CURRENT_START = 503
	REG_CURRENT_LEN   = 21
	REG_POWER_START   = 621
	REG_POWER_LEN     = 17
	REG_ENERGY_START  = 848
	REG_ENERGY_LEN    = 26

	switchMask = 0x8000
)

// 电压段：
// 这一段只放最核心的输入/输出电压，属于列头柜供配电拓扑里的第一层概览量。
var voltagePointSpecs = []blockPointSpec{
	{Index: 0, Field: "UA1", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "市电总输入A"},
	{Index: 1, Field: "UB1", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "市电总输入B"},
	{Index: 2, Field: "UC1", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "市电总输入C"},
	{Index: 3, Field: "Uups", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "UPS输出"},
}

// 电流段：
// 前半段是市电输入与 UPS 输入/输出相电流，后半段扩展到各路 PDU 支路电流。
// 用同一个数组描述，是因为协议文档本身就是一段连续寄存器块。
var currentPointSpecs = []blockPointSpec{
	{Index: 0, Field: "MainsACurr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电输入A相电流"},
	{Index: 1, Field: "MainsBCurr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电输入B相电流"},
	{Index: 2, Field: "MainsCCurr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电输入C相电流"},
	{Index: 3, Field: "UPSIC", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "UPS输入总电流"},
	{Index: 4, Field: "UPSACurr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "UPS输出A相电流"},
	{Index: 5, Field: "UPSBCurr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "UPS输出B相电流"},
	{Index: 6, Field: "UPSCCurr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "UPS输出C相电流"},
	{Index: 7, Field: "MainsPdu1Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电PDU-1电流"},
	{Index: 8, Field: "MainsPdu2Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电PDU-2电流"},
	{Index: 9, Field: "MainsPdu3Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电PDU-3电流"},
	{Index: 10, Field: "MainsPdu4Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电PDU-4电流"},
	{Index: 11, Field: "MainsPdu5Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电PDU-5电流"},
	{Index: 12, Field: "MainsPdu6Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电PDU-6电流"},
	{Index: 13, Field: "MainsPdu7Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "市电PDU-7电流"},
	{Index: 14, Field: "UpsPdu1Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "U电PDU-1电流"},
	{Index: 15, Field: "UpsPdu2Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "U电PDU-2电流"},
	{Index: 16, Field: "UpsPdu3Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "U电PDU-3电流"},
	{Index: 17, Field: "UpsPdu4Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "U电PDU-4电流"},
	{Index: 18, Field: "UpsPdu5Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "U电PDU-5电流"},
	{Index: 19, Field: "UpsPdu6Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "U电PDU-6电流"},
	{Index: 20, Field: "UpsPdu7Curr", Scale: 0.1, Decimals: 1, RW: "R", Unit: "A", Label: "U电PDU-7电流"},
}

// 功率段：
// 结构与电流段基本平行，先三相总功率，再展开到各个市电/U 电 PDU 的分路功率。
var powerPointSpecs = []blockPointSpec{
	{Index: 0, Field: "MainsPA", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电输出A相功率"},
	{Index: 1, Field: "MainsPB", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电输出B相功率"},
	{Index: 2, Field: "MainsPC", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电输出C相功率"},
	{Index: 3, Field: "MainsPdu1P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电PDU1功率"},
	{Index: 4, Field: "MainsPdu2P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电PDU2功率"},
	{Index: 5, Field: "MainsPdu3P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电PDU3功率"},
	{Index: 6, Field: "MainsPdu4P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电PDU4功率"},
	{Index: 7, Field: "MainsPdu5P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电PDU5功率"},
	{Index: 8, Field: "MainsPdu6P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电PDU6功率"},
	{Index: 9, Field: "MainsPdu7P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "市电PDU7功率"},
	{Index: 10, Field: "UpsPdu1P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "U电PDU1功率"},
	{Index: 11, Field: "UpsPdu2P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "U电PDU2功率"},
	{Index: 12, Field: "UpsPdu3P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "U电PDU3功率"},
	{Index: 13, Field: "UpsPdu4P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "U电PDU4功率"},
	{Index: 14, Field: "UpsPdu5P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "U电PDU5功率"},
	{Index: 15, Field: "UpsPdu6P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "U电PDU6功率"},
	{Index: 16, Field: "UpsPdu7P", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kW", Label: "U电PDU7功率"},
}

// 电能段：
// 电能值跨 2 个寄存器，且文档中的寄存器顺序和业务展示顺序并不完全一致，
// 因此这里显式记录真实寄存器地址，而不是只记块内下标。
var energyPointSpecs = []energyPointSpec{
	{Register: 854, Field: "MainsEPA", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电输出A相电能"},
	{Register: 856, Field: "MainsEPB", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电输出B相电能"},
	{Register: 858, Field: "MainsEPC", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电输出C相电能"},
	{Register: 860, Field: "MainsPdu1EP", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电PDU1电能"},
	{Register: 848, Field: "MainsPdu2EP", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电PDU2电能"},
	{Register: 850, Field: "MainsPdu3EP", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电PDU3电能"},
	{Register: 866, Field: "MainsPdu4EP", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电PDU4电能"},
	{Register: 868, Field: "MainsPdu5EP", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电PDU5电能"},
	{Register: 870, Field: "MainsPdu6EP", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电PDU6电能"},
	{Register: 872, Field: "MainsPdu7EP", Scale: 0.1, Decimals: 1, RW: "R", Unit: "kWh", Label: "市电PDU7电能"},
}

// 开关段：
// 这一段不是工程量，而是状态位集合。这里只挑监控系统真正关心的总开关和各路 PDU 开关。
var switchPointSpecs = []switchPointSpec{
	{Index: 0, Field: "MSS", Label: "市电总输入开关状态"},
	{Index: 3, Field: "MainsPdu1Switch", Label: "市电PDU1开关状态"},
	{Index: 4, Field: "MainsPdu2Switch", Label: "市电PDU2开关状态"},
	{Index: 5, Field: "MainsPdu3Switch", Label: "市电PDU3开关状态"},
	{Index: 6, Field: "MainsPdu4Switch", Label: "市电PDU4开关状态"},
	{Index: 7, Field: "MainsPdu5Switch", Label: "市电PDU5开关状态"},
	{Index: 8, Field: "MainsPdu6Switch", Label: "市电PDU6开关状态"},
	{Index: 9, Field: "MainsPdu7Switch", Label: "市电PDU7开关状态"},
	{Index: 10, Field: "UpsPdu1Switch", Label: "U电PDU1开关状态"},
	{Index: 11, Field: "UpsPdu2Switch", Label: "U电PDU2开关状态"},
	{Index: 12, Field: "UpsPdu3Switch", Label: "U电PDU3开关状态"},
	{Index: 13, Field: "UpsPdu4Switch", Label: "U电PDU4开关状态"},
	{Index: 14, Field: "UpsPdu5Switch", Label: "U电PDU5开关状态"},
	{Index: 15, Field: "UpsPdu6Switch", Label: "U电PDU6开关状态"},
	{Index: 16, Field: "UpsPdu7Switch", Label: "U电PDU7开关状态"},
}

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
	points := readAllPoints(cfg.DeviceAddress)

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
func readAllPoints(devAddr int) []DriverPoint {
	points := make([]DriverPoint, 0, 67)
	addr := byte(devAddr)

	// 按电压、电流、功率、电能、开关五类组织输出，方便维护时按业务类别定位问题。
	points = appendBlockPoints(points, addr, REG_VOLTAGE_START, REG_VOLTAGE_LEN, voltagePointSpecs)
	points = appendBlockPoints(points, addr, REG_CURRENT_START, REG_CURRENT_LEN, currentPointSpecs)
	points = appendBlockPoints(points, addr, REG_POWER_START, REG_POWER_LEN, powerPointSpecs)
	points = appendEnergyPoints(points, addr)
	points = appendSwitchPoints(points, addr)

	return points
}

func appendBlockPoints(points []DriverPoint, devAddr byte, startReg uint16, count uint16, specs []blockPointSpec) []DriverPoint {
	// 普通数值点的读取和映射规律稳定，因此统一通过这层模板化处理。
	values := readMultipleRegs(devAddr, startReg, count)
	if values == nil {
		return points
	}

	for _, spec := range specs {
		if spec.Index < 0 || spec.Index >= len(values) {
			continue
		}
		points = append(points, makeScaledPoint(spec, uint32(values[spec.Index])))
	}
	return points
}

func appendEnergyPoints(points []DriverPoint, devAddr byte) []DriverPoint {
	// 电能点位共享一段寄存器，但每个点都跨 2 个寄存器，需要单独解码。
	values := readMultipleRegs(devAddr, REG_ENERGY_START, REG_ENERGY_LEN)
	if values == nil {
		return points
	}

	for _, spec := range energyPointSpecs {
		raw, ok := readU32(values, REG_ENERGY_START, spec.Register)
		if !ok {
			continue
		}
		points = append(points, makeEnergyPoint(spec, raw))
	}
	return points
}

func appendSwitchPoints(points []DriverPoint, devAddr byte) []DriverPoint {
	// 开关状态与普通工程量点不同，因此单独保留一条清晰的处理路径。
	values := readMultipleRegs(devAddr, REG_SWITCH_START, REG_SWITCH_LEN)
	if values == nil {
		return points
	}

	for _, spec := range switchPointSpecs {
		if spec.Index < 0 || spec.Index >= len(values) {
			continue
		}
		points = append(points, makeSwitchPoint(spec, values[spec.Index]))
	}
	return points
}

func makeScaledPoint(spec blockPointSpec, raw uint32) DriverPoint {
	return makeNumericPoint(spec.Field, float64(raw)*spec.Scale, spec.Decimals, spec.RW, spec.Unit, spec.Label)
}

func makeEnergyPoint(spec energyPointSpec, raw uint32) DriverPoint {
	return makeNumericPoint(spec.Field, float64(raw)*spec.Scale, spec.Decimals, spec.RW, spec.Unit, spec.Label)
}

func makeSwitchPoint(spec switchPointSpec, raw uint16) DriverPoint {
	// 当前协议只使用最高位表示开关状态，因此这里做一次显式掩码。
	return DriverPoint{
		FieldName: spec.Field,
		Value:     strconv.Itoa(int(raw & switchMask)),
		RW:        "R",
		Unit:      "",
		Label:     spec.Label,
	}
}

func makeNumericPoint(field string, value float64, decimals int, rw, unit, label string) DriverPoint {
	return DriverPoint{
		FieldName: field,
		Value:     tinydrv.FormatFloat(value, decimals),
		RW:        rw,
		Unit:      unit,
		Label:     label,
	}
}

func readU32(values []uint16, startReg uint16, targetReg uint16) (uint32, bool) {
	// 通过“真实寄存器地址 -> 当前切片偏移”的换算，
	// 调用方可以继续用协议文档里的地址思考，而不是被数组下标绑住。
	idx := int(targetReg - startReg)
	if idx < 0 || idx+1 >= len(values) {
		return 0, false
	}
	return uint32(values[idx])<<16 | uint32(values[idx+1]), true
}

// =============================================================================
// 【固定不变】Modbus TCP 通信函数
// =============================================================================
func readMultipleRegs(devAddr byte, startReg uint16, count uint16) []uint16 {
	// 列头柜走标准 Modbus TCP 03 读保持寄存器，无需额外兼容分支。
	values, err := modbustcp.ReadRegisters(tcpTransceive, devAddr, FUNC_CODE_READ, startReg, count, 1000, 256)
	if err != nil {
		return nil
	}
	return values
}

func tcpTransceive(req []byte, resp []byte, timeoutMs int) int {
	return hostio.TransceiveInto(callTCPTransceive, req, resp, timeoutMs)
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
	}
}

func main() {}
