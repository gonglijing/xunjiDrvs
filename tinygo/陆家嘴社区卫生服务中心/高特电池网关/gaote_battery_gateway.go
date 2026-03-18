// =============================================================================
// 高特电池网关 - Modbus RTU 驱动（陆家嘴社区卫生服务中心）
// =============================================================================
//
// 协议类型: Modbus RTU
// 功能码: 0x04 (INPUT_REGISTER)
//
// 主要测点:
//   - 电池1~40电压: U01~U40, 地址400~439, 表达式 v/1000
//   - 电池1~40温度: T01~T40, 地址800~839, 表达式 v/10-40
//   - 电池1~40内阻: IR01~IR40, 地址1200~1239, 表达式 v/1000
//   - 组电压: TU, 地址0, 长度2, 表达式 v/10
//   - 组电流: TI, 地址2, 长度2, 表达式 v/1000
//   - 环境温度: T, 地址4, 长度1, 表达式 v/10-40
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

type indexedPointMeta struct {
	Field    string
	Label    string
	Decimals int
	RW       string
	Unit     string
}

// =============================================================================
// 【用户修改】驱动版本
// =============================================================================
const (
	DriverVersion    = "1.0.0"
	DriverProductKey = "ljzchc_gaote_battery_gateway"
)

// =============================================================================
// 【用户修改】协议定义
// =============================================================================
const (
	FUNC_CODE_READ_INPUT = 0x04

	REG_GROUP_START = 0
	REG_GROUP_LEN   = 5

	REG_U_START = 400
	REG_U_LEN   = 40

	REG_T_START = 800
	REG_T_LEN   = 40

	REG_IR_START = 1200
	REG_IR_LEN   = 40
)

var voltagePointMetas = buildIndexedPointMetas("U", "电池", "#电压", REG_U_LEN, 3, "R", "V")
var temperaturePointMetas = buildIndexedPointMetas("T", "电池", "#温度", REG_T_LEN, 1, "R", "℃")
var resistancePointMetas = buildIndexedPointMetas("IR", "电池", "#内阻", REG_IR_LEN, 3, "R", "Ω")

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
	points := make([]DriverPoint, 0, 123)

	// 组级参数: TU(2寄存器), TI(2寄存器), T(1寄存器)
	if values := readMultipleRegs(byte(devAddr), REG_GROUP_START, REG_GROUP_LEN, debug); values != nil && len(values) >= 5 {
		tuRaw := combineTwoRegs(values[0], values[1])
		tiRaw := combineTwoRegs(values[2], values[3])
		tRaw := int64(values[4])

		points = append(points, makePointValue("TU", float64(tuRaw)/10.0, 1, "R", "V", "组电压"))
		points = append(points, makePointValue("TI", float64(tiRaw)/1000.0, 3, "R", "A", "组电流"))
		points = append(points, makePointValue("T", float64(tRaw)/10.0-40.0, 1, "R", "℃", "环境温度"))
	}

	// 电池1~40电压 U01~U40: v/1000
	if values := readMultipleRegs(byte(devAddr), REG_U_START, REG_U_LEN, debug); values != nil && len(values) >= REG_U_LEN {
		points = appendIndexedPoints(points, values, voltagePointMetas, func(raw uint16) float64 {
			return float64(raw) / 1000.0
		})
	}

	// 电池1~40温度 T01~T40: v/10-40
	if values := readMultipleRegs(byte(devAddr), REG_T_START, REG_T_LEN, debug); values != nil && len(values) >= REG_T_LEN {
		points = appendIndexedPoints(points, values, temperaturePointMetas, func(raw uint16) float64 {
			return float64(raw)/10.0 - 40.0
		})
	}

	// 电池1~40内阻 IR01~IR40: v/1000
	if values := readMultipleRegs(byte(devAddr), REG_IR_START, REG_IR_LEN, debug); values != nil && len(values) >= REG_IR_LEN {
		points = appendIndexedPoints(points, values, resistancePointMetas, func(raw uint16) float64 {
			return float64(raw) / 1000.0
		})
	}

	return points
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

func combineTwoRegs(high uint16, low uint16) int64 {
	v := (uint32(high) << 16) | uint32(low)
	return int64(int32(v))
}

func appendIndexedPoints(
	points []DriverPoint,
	values []uint16,
	metas []indexedPointMeta,
	transform func(uint16) float64,
) []DriverPoint {
	for index, meta := range metas {
		points = append(points, makePointValue(
			meta.Field,
			transform(values[index]),
			meta.Decimals,
			meta.RW,
			meta.Unit,
			meta.Label,
		))
	}
	return points
}

func buildIndexedPointMetas(prefix string, labelPrefix string, labelSuffix string, count int, decimals int, rw string, unit string) []indexedPointMeta {
	metas := make([]indexedPointMeta, 0, count)
	for i := 1; i <= count; i++ {
		indexText := strconv.Itoa(i)
		metas = append(metas, indexedPointMeta{
			Field:    prefix + twoDigit(i),
			Label:    labelPrefix + indexText + labelSuffix,
			Decimals: decimals,
			RW:       rw,
			Unit:     unit,
		})
	}
	return metas
}

func twoDigit(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// =============================================================================
// 【固定不变】Modbus RTU 通信函数
// =============================================================================

func readMultipleRegs(devAddr byte, startReg uint16, count uint16, debug bool) []uint16 {
	values, err := modbusrtu.ReadRegisters(serialTransceive, devAddr, FUNC_CODE_READ_INPUT, startReg, count, 1000, debug, 24, tinydrv.Logf)
	if err != nil {
		return nil
	}

	return values
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
