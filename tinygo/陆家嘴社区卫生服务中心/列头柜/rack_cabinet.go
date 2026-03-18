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

	pdk "github.com/extism/go-pdk"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/modbustcp"
	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/tinydrv"
)

// =============================================================================
// 【固定不变】Host 函数声明
// =============================================================================
//
//go:wasmimport extism:host/user tcp_transceive
func tcp_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

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
	Register uint16
	Field    string
	Scale    float64
	Decimals int
	RW       string
	Unit     string
	Label    string
}

type switchPointSpec struct {
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

var voltagePointSpecs = []blockPointSpec{
	{Index: 0, Field: "UA1", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "市电总输入A"},
	{Index: 1, Field: "UB1", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "市电总输入B"},
	{Index: 2, Field: "UC1", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "市电总输入C"},
	{Index: 3, Field: "Uups", Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "UPS输出"},
}

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
			outputJSON(ErrorResponse{Success: false, Error: "panic"})
		}
	}()

	cfg := getConfig()
	points := readAllPoints(cfg.DeviceAddress)

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
func readAllPoints(devAddr int) []DriverPoint {
	points := make([]DriverPoint, 0, 67)
	addr := byte(devAddr)

	points = appendBlockPoints(points, addr, REG_VOLTAGE_START, REG_VOLTAGE_LEN, voltagePointSpecs)
	points = appendBlockPoints(points, addr, REG_CURRENT_START, REG_CURRENT_LEN, currentPointSpecs)
	points = appendBlockPoints(points, addr, REG_POWER_START, REG_POWER_LEN, powerPointSpecs)
	points = appendEnergyPoints(points, addr)
	points = appendSwitchPoints(points, addr)

	return points
}

func appendBlockPoints(points []DriverPoint, devAddr byte, startReg uint16, count uint16, specs []blockPointSpec) []DriverPoint {
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
		Value:     formatFloat(value, decimals),
		RW:        rw,
		Unit:      unit,
		Label:     label,
	}
}

func readU32(values []uint16, startReg uint16, targetReg uint16) (uint32, bool) {
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
	req := buildReadRequest(devAddr, startReg, count)
	resp := make([]byte, 256)

	n := tcpTransceive(req, resp, 1000)
	if n < 9 {
		return nil
	}

	values, err := parseReadResponse(resp[:n], devAddr)
	if err != nil || len(values) < int(count) {
		return nil
	}
	return values
}

func tcpTransceive(req []byte, resp []byte, timeoutMs int) int {
	if len(req) == 0 || len(resp) == 0 {
		return 0
	}

	reqMem := pdk.AllocateBytes(req)
	defer reqMem.Free()
	respMem := pdk.Allocate(len(resp))
	defer respMem.Free()

	n := int(tcp_transceive(
		reqMem.Offset(), uint64(len(req)),
		respMem.Offset(), uint64(len(resp)),
		uint64(timeoutMs),
	))
	if n <= 0 {
		return n
	}
	if n > len(resp) {
		n = len(resp)
	}

	mem := pdk.NewMemory(respMem.Offset(), uint64(n))
	mem.Load(resp[:n])
	return n
}

func buildReadRequest(addr byte, startReg uint16, count uint16) []byte {
	return modbustcp.BuildReadRequest(addr, FUNC_CODE_READ, startReg, count)
}

func parseReadResponse(data []byte, addr byte) ([]uint16, error) {
	return modbustcp.ParseReadResponse(data, addr, FUNC_CODE_READ)
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

func formatFloat(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

func outputJSON(v interface{}) {
	tinydrv.OutputJSON(v)
}

func main() {}
