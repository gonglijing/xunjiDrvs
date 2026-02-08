// =============================================================================
// 科士达 UPS - Modbus TCP 驱动
// =============================================================================
//
// 设备点表:
//   - 电池容量: FC=03, 地址=100, 长度=1, 缩放=0.1, 1位小数
//   - 电池剩余时间: FC=03, 地址=101, 长度=1, 缩放=1, 整数
//   - 输入频率: FC=03, 地址=109, 长度=1, 缩放=0.1, 1位小数
//   - R相输入电压: FC=03, 地址=110, 长度=1, 缩放=0.1, 1位小数
//   - S相输入电压: FC=03, 地址=111, 长度=1, 缩放=0.1, 1位小数
//   - T相输入电压: FC=03, 地址=112, 长度=1, 缩放=0.1, 1位小数
//   - 输出频率: FC=03, 地址=119, 长度=1, 缩放=0.1, 1位小数
//   - R相输出电压: FC=03, 地址=120, 长度=1, 缩放=0.1, 1位小数
//   - S相输出电压: FC=03, 地址=121, 长度=1, 缩放=0.1, 1位小数
//   - T相输出电压: FC=03, 地址=122, 长度=1, 缩放=0.1, 1位小数
//   - R相负载率: FC=03, 地址=123, 长度=1, 缩放=1, 整数
//   - S相负载率: FC=03, 地址=124, 长度=1, 缩放=1, 整数
//   - T相负载率: FC=03, 地址=125, 长度=1, 缩放=1, 整数
//
// Host 提供: tcp_transceive
//
// =============================================================================
package main

import (
	"encoding/binary"
	"encoding/json"
	"strconv"
	"strings"

	pdk "github.com/extism/go-pdk"
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
	DeviceAddress int    `json:"device_address"` // Modbus 从站地址
	FuncName      string `json:"func_name"`      // "read" | "write"
	FieldName     string `json:"field_name"`     // 可写字段名
	Value         string `json:"value"`          // 写操作的值
}

// =============================================================================
// 【用户修改】驱动版本
// =============================================================================
const DriverVersion = "1.0.0"

// =============================================================================
// 【用户修改】点表定义
// =============================================================================
// 根据实际设备修改以下寄存器地址
const (
	// 电池参数
	REG_BATTERY_CAPACITY    = 100 // 电池容量 (%, v/10)
	REG_BATTERY_REMAIN_TIME = 101 // 电池剩余时间 (min)

	// 输入参数
	REG_INPUT_FREQUENCY = 109 // 输入频率 (Hz, v/10)
	REG_INPUT_VOLTAGE_R = 110 // R相输入电压 (V, v/10)
	REG_INPUT_VOLTAGE_S = 111 // S相输入电压 (V, v/10)
	REG_INPUT_VOLTAGE_T = 112 // T相输入电压 (V, v/10)

	// 输出参数
	REG_OUTPUT_FREQUENCY = 119 // 输出频率 (Hz, v/10)
	REG_OUTPUT_VOLTAGE_R = 120 // R相输出电压 (V, v/10)
	REG_OUTPUT_VOLTAGE_S = 121 // S相输出电压 (V, v/10)
	REG_OUTPUT_VOLTAGE_T = 122 // T相输出电压 (V, v/10)
	REG_LOAD_PERCENT_R   = 123 // R相负载率 (%)
	REG_LOAD_PERCENT_S   = 124 // S相负载率 (%)
	REG_LOAD_PERCENT_T   = 125 // T相负载率 (%)

	// 功能码
	FUNC_CODE_READ = 0x03 // 读保持寄存器
)

// =============================================================================
// 【用户修改】点表配置
// =============================================================================
// 定义所有需要读取的测点
// 根据寄存器地址分组，实现批量读取优化
var pointConfig = []PointConfig{
	// 电池参数 (单个读取)
	{Field: "battery_capacity", Address: REG_BATTERY_CAPACITY, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "%", Label: "电池容量"},
	{Field: "battery_remain_time", Address: REG_BATTERY_REMAIN_TIME, Length: 1, Scale: 1, Decimals: 0, RW: "R", Unit: "min", Label: "电池剩余时间"},

	// 输入参数 (109~112, 连续4个寄存器)
	{Field: "input_frequency", Address: REG_INPUT_FREQUENCY, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "Hz", Label: "输入频率"},
	{Field: "input_voltage_r", Address: REG_INPUT_VOLTAGE_R, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "R相输入电压"},
	{Field: "input_voltage_s", Address: REG_INPUT_VOLTAGE_S, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "S相输入电压"},
	{Field: "input_voltage_t", Address: REG_INPUT_VOLTAGE_T, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "T相输入电压"},

	// 输出参数 (119~125, 连续7个寄存器)
	{Field: "output_frequency", Address: REG_OUTPUT_FREQUENCY, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "Hz", Label: "输出频率"},
	{Field: "output_voltage_r", Address: REG_OUTPUT_VOLTAGE_R, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "R相输出电压"},
	{Field: "output_voltage_s", Address: REG_OUTPUT_VOLTAGE_S, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "S相输出电压"},
	{Field: "output_voltage_t", Address: REG_OUTPUT_VOLTAGE_T, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "V", Label: "T相输出电压"},
	{Field: "load_percent_r", Address: REG_LOAD_PERCENT_R, Length: 1, Scale: 1, Decimals: 0, RW: "R", Unit: "%", Label: "R相负载率"},
	{Field: "load_percent_s", Address: REG_LOAD_PERCENT_S, Length: 1, Scale: 1, Decimals: 0, RW: "R", Unit: "%", Label: "S相负载率"},
	{Field: "load_percent_t", Address: REG_LOAD_PERCENT_T, Length: 1, Scale: 1, Decimals: 0, RW: "R", Unit: "%", Label: "T相负载率"},
}

// 点表配置结构
type PointConfig struct {
	Field    string  // 字段名
	Address  uint16  // 寄存器地址
	Length   uint16  // 寄存器数量
	Scale    float64 // 缩放系数
	Decimals int     // 有效小数位数
	RW       string  // 读写属性
	Unit     string  // 单位
	Label    string  // 显示标签
}

// =============================================================================
// 【固定不变】驱动入口
// =============================================================================
//
//go:wasmexport handle
func handle() int32 {
	defer func() {
		if r := recover(); r != nil {
			outputJSON(map[string]interface{}{"success": false, "error": "panic"})
		}
	}()

	cfg := getConfig()

	// 读操作 - 读取所有监控参数
	points := readAllUPS(cfg.DeviceAddress)

	outputJSON(map[string]interface{}{
		"success": true,
		"points":  points,
	})
	return 0
}

// =============================================================================
// 【固定不变】描述可写字段
// =============================================================================
//
//go:wasmexport describe
func describe() int32 {
	outputJSON(map[string]interface{}{
		"success": true,
		"data":    map[string]string{},
	})
	return 0
}

// =============================================================================
// 【固定不变】驱动版本
// =============================================================================
//
//go:wasmexport version
func version() int32 {
	outputJSON(map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"version": DriverVersion,
		},
	})
	return 0
}

// =============================================================================
// 【用户修改】读取所有测点
// =============================================================================
// 根据寄存器地址分组批量读取
func readAllUPS(devAddr int) []map[string]interface{} {
	points := make([]map[string]interface{}, 0)

	// 1. 电池参数 (单个读取)
	// ====================
	// 电池容量
	if val := readSingleReg(byte(devAddr), REG_BATTERY_CAPACITY); val >= 0 {
		points = append(points, makePoint("battery_capacity", int(val), 0.1, 1, "R", "%", "电池容量"))
	}
	// 电池剩余时间
	if val := readSingleReg(byte(devAddr), REG_BATTERY_REMAIN_TIME); val >= 0 {
		points = append(points, makePoint("battery_remain_time", int(val), 1, 0, "R", "min", "电池剩余时间"))
	}

	// 2. 输入参数 (109~112, 批量读取)
	// ====================
	if values := readMultipleRegs(byte(devAddr), REG_INPUT_FREQUENCY, 4); values != nil {
		points = append(points, makePoint("input_frequency", int(values[0]), 0.1, 1, "R", "Hz", "输入频率"))
		points = append(points, makePoint("input_voltage_r", int(values[1]), 0.1, 1, "R", "V", "R相输入电压"))
		points = append(points, makePoint("input_voltage_s", int(values[2]), 0.1, 1, "R", "V", "S相输入电压"))
		points = append(points, makePoint("input_voltage_t", int(values[3]), 0.1, 1, "R", "V", "T相输入电压"))
	}

	// 3. 输出参数 (119~125, 批量读取)
	// ====================
	if values := readMultipleRegs(byte(devAddr), REG_OUTPUT_FREQUENCY, 7); values != nil {
		points = append(points, makePoint("output_frequency", int(values[0]), 0.1, 1, "R", "Hz", "输出频率"))
		points = append(points, makePoint("output_voltage_r", int(values[1]), 0.1, 1, "R", "V", "R相输出电压"))
		points = append(points, makePoint("output_voltage_s", int(values[2]), 0.1, 1, "R", "V", "S相输出电压"))
		points = append(points, makePoint("output_voltage_t", int(values[3]), 0.1, 1, "R", "V", "T相输出电压"))
		points = append(points, makePoint("load_percent_r", int(values[4]), 1, 0, "R", "%", "R相负载率"))
		points = append(points, makePoint("load_percent_s", int(values[5]), 1, 0, "R", "%", "S相负载率"))
		points = append(points, makePoint("load_percent_t", int(values[6]), 1, 0, "R", "%", "T相负载率"))
	}

	return points
}

// 创建测点数据
func makePoint(field string, rawVal int, scale float64, decimals int, rw, unit, label string) map[string]interface{} {
	realVal := float64(rawVal) * scale
	return map[string]interface{}{
		"field_name": field,
		"value":      formatFloat(realVal, decimals),
		"rw":         rw,
		"unit":       unit,
		"label":      label,
	}
}

// =============================================================================
// 【固定不变】Modbus TCP 通信函数
// =============================================================================

// 读取单个寄存器
func readSingleReg(devAddr byte, regAddr uint16) int {
	values := readMultipleRegs(devAddr, regAddr, 1)
	if values == nil || len(values) < 1 {
		return -1
	}
	return int(values[0])
}

// 批量读取寄存器 (优化: 减少通信次数)
func readMultipleRegs(devAddr byte, startReg uint16, count uint16) []int16 {
	req := buildReadRequest(devAddr, startReg, count)
	resp := make([]byte, 64)

	n := tcpTransceive(req, resp, 1000)
	if n < 7 {
		return nil
	}

	values, err := parseReadResponse(resp[:n], devAddr)
	if err != nil || len(values) < int(count) {
		return nil
	}

	result := make([]int16, count)
	for i := 0; i < int(count); i++ {
		result[i] = int16(values[i])
	}
	return result
}

// TCP 发送接收 (通用)
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

// 构建 Modbus TCP 读请求帧 (通用)
func buildReadRequest(addr byte, startReg uint16, count uint16) []byte {
	// MBAP 头 (7字节) + PDU (5字节) = 12字节
	mbap := make([]byte, 12)
	mbap[0] = 0x00 // 事务标识符
	mbap[1] = 0x01
	mbap[2] = 0x00 // 协议标识符
	mbap[3] = 0x00
	mbap[4] = 0x00 // 长度高字节
	mbap[5] = 0x06 // 长度低字节 (6字节)
	mbap[6] = addr // 单元标识符 (从站地址)

	// PDU
	mbap[7] = FUNC_CODE_READ      // 功能码
	mbap[8] = byte(startReg >> 8) // 起始地址高字节
	mbap[9] = byte(startReg)      // 起始地址低字节
	mbap[10] = byte(count >> 8)   // 数量高字节
	mbap[11] = byte(count)        // 数量低字节

	return mbap
}

// 解析 Modbus TCP 读响应 (通用)
func parseReadResponse(data []byte, addr byte) ([]uint16, error) {
	// 跳过 MBAP 头 (7字节)
	pdu := data[6:]

	if len(pdu) < 3 {
		return nil, errf("响应数据不完整")
	}

	if pdu[0] != addr || pdu[1] != FUNC_CODE_READ {
		return nil, errf("响应地址或功能码不匹配")
	}

	byteCount := int(pdu[2])
	if len(pdu) < 3+byteCount {
		return nil, errf("响应数据长度不足")
	}

	values := make([]uint16, byteCount/2)
	for i := 0; i < len(values); i++ {
		values[i] = binary.BigEndian.Uint16(pdu[3+i*2:])
	}

	return values, nil
}

// =============================================================================
// 【固定不变】工具函数
// =============================================================================

// 获取配置 (通用)
func getConfig() DriverConfig {
	def := DriverConfig{DeviceAddress: 1, FuncName: "read"}
	var envelope struct {
		Config map[string]string `json:"config"`
	}
	if err := pdk.InputJSON(&envelope); err != nil {
		return def
	}

	cfg := def
	if v := strings.TrimSpace(envelope.Config["device_address"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DeviceAddress = n
		}
	}
	if v := strings.TrimSpace(envelope.Config["func_name"]); v != "" {
		cfg.FuncName = v
	}
	if v := strings.TrimSpace(envelope.Config["field_name"]); v != "" {
		cfg.FieldName = v
	}
	if v := strings.TrimSpace(envelope.Config["value"]); v != "" {
		cfg.Value = v
	}
	return cfg
}

// 格式化浮点数 (通用)
func formatFloat(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

// 错误类型 (通用)
type simpleErr string

func (e simpleErr) Error() string { return string(e) }
func errf(s string) error         { return simpleErr(s) }

// 输出 JSON (通用)
func outputJSON(v interface{}) {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		b = []byte(`{"success":false,"error":"encode failed"}`)
	}
	pdk.Output(b)
}

func main() {}
