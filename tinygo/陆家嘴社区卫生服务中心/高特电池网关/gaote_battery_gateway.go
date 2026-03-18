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

// 本地包装层只负责把 wasmimport 暴露为普通函数值。
func callSerialTransceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64 {
	return serial_transceive(wPtr, wSize, rPtr, rCap, timeoutMs)
}

// =============================================================================
// 【固定不变】配置结构（网关传入）
// =============================================================================
type DriverConfig = tinydrv.DriverConfig

type DriverPoint = tinydrv.Point

// indexedPointMeta 用于描述“第 N 个电池点位最终应该长什么样”，
// 这样批量点位就不需要手写 40 组几乎相同的结构。
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

// 单体电压段：协议把 40 节电池电压按连续地址排布成一整段，
// 因此这里一次性生成 U01~U40 元数据，维护时只需关注总节数和命名规则。
var voltagePointMetas = buildIndexedPointMetas("U", "电池", "#电压", REG_U_LEN, 3, "R", "V")

// 单体温度段：字段名仍保持 T01~T40，与现场点表和历史系统中的叫法一致，
// 这样排查单节异常时，可以直接按“第几节电池温度”去比对。
var temperaturePointMetas = buildIndexedPointMetas("T", "电池", "#温度", REG_T_LEN, 1, "R", "℃")

// 单体内阻段：内阻与电压/温度同样都是 40 节一一对应，
// 独立成单独元数据数组后，后续若某一段寄存器发生偏移，只需要改对应一段。
var resistancePointMetas = buildIndexedPointMetas("IR", "电池", "#内阻", REG_IR_LEN, 3, "R", "Ω")

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
	points := readAllPoints(cfg.DeviceAddress, cfg.Debug)

	tinydrv.OutputHandleSuccess(DriverProductKey, points)
	return 0
}

func writeNotSupported(fieldName string) int32 {
	errText := "高特电池网关驱动当前仅支持读取，不支持写入"
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
		"transport": "serial",
		"protocol":  "modbus_rtu",
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
		"transport": "serial",
		"protocol":  "modbus_rtu",
	})
	return 0
}

// =============================================================================
// 【用户修改】读取所有测点
// =============================================================================
func readAllPoints(devAddr int, debug bool) []DriverPoint {
	points := make([]DriverPoint, 0, 123)

	// 组级参数和单体参数按协议文档分块读取，结构更接近现场点表。
	if values := readMultipleRegs(byte(devAddr), REG_GROUP_START, REG_GROUP_LEN, debug); values != nil && len(values) >= 5 {
		tuRaw := combineTwoRegs(values[0], values[1])
		tiRaw := combineTwoRegs(values[2], values[3])
		tRaw := int64(values[4])

		points = append(points, makePointValue("TU", float64(tuRaw)/10.0, 1, "R", "V", "组电压"))
		points = append(points, makePointValue("TI", float64(tiRaw)/1000.0, 3, "R", "A", "组电流"))
		points = append(points, makePointValue("T", float64(tRaw)/10.0-40.0, 1, "R", "℃", "环境温度"))
	}

	// 单体电压段：
	// 这一段是最基础的电池健康数据，每个寄存器对应一节电池的瞬时电压。
	if values := readMultipleRegs(byte(devAddr), REG_U_START, REG_U_LEN, debug); values != nil && len(values) >= REG_U_LEN {
		points = appendIndexedPoints(points, values, voltagePointMetas, func(raw uint16) float64 {
			return float64(raw) / 1000.0
		})
	}

	// 单体温度段：
	// 协议原始值需要先除以 10 再减 40，这是设备文档给出的补码式偏移换算。
	if values := readMultipleRegs(byte(devAddr), REG_T_START, REG_T_LEN, debug); values != nil && len(values) >= REG_T_LEN {
		points = appendIndexedPoints(points, values, temperaturePointMetas, func(raw uint16) float64 {
			return float64(raw)/10.0 - 40.0
		})
	}

	// 单体内阻段：
	// 这组数据通常和电压一起用来判断电池老化，因此保留与单体编号完全一致的输出顺序。
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
	// 组电压、组电流属于 32 位值，这里按“高寄存器在前”的常见 Modbus 方式组合。
	v := (uint32(high) << 16) | uint32(low)
	return int64(int32(v))
}

func appendIndexedPoints(
	points []DriverPoint,
	values []uint16,
	metas []indexedPointMeta,
	transform func(uint16) float64,
) []DriverPoint {
	// 统一的批量造点逻辑，让调用方只描述“如何把单个寄存器转成数值”。
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
	// 元数据在初始化时一次性生成，可避免运行时反复拼字段名和标签。
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
	// 现场命名要求固定宽度，例如 U01/T03/IR40。
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// =============================================================================
// 【固定不变】Modbus RTU 通信函数
// =============================================================================

func readMultipleRegs(devAddr byte, startReg uint16, count uint16, debug bool) []uint16 {
	// 高特网关所有批量点位都走 0x04 读输入寄存器，因此这里保持统一入口。
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
	return tinydrv.ParseDriverConfig(DriverConfig{DeviceAddress: 1, FuncName: "read"})
}

func main() {}
