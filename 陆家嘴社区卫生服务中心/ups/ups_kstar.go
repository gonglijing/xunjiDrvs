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
func readAllUPS(devAddr int) []map[string]interface{} {
	points := make([]map[string]interface{}, 0)

	if values := readMultipleRegs(byte(devAddr), REG_OUTPUT_FREQUENCY, 7); values != nil {
		points = append(points, makePoint("OUR", int(values[1]), 0.1, 1, "R", "V", "R相输出电压"))
		points = append(points, makePoint("OUS", int(values[2]), 0.1, 1, "R", "V", "S相输出电压"))
		points = append(points, makePoint("OUT", int(values[3]), 0.1, 1, "R", "V", "T相输出电压"))
		points = append(points, makePoint("OH", int(values[0]), 0.1, 1, "R", "Hz", "输出频率"))
		points = append(points, makePoint("loadR", int(values[4]), 1, 0, "R", "%", "R相负载率"))
		points = append(points, makePoint("loadS", int(values[5]), 1, 0, "R", "%", "S相负载率"))
		points = append(points, makePoint("loadT", int(values[6]), 1, 0, "R", "%", "T相负载率"))
	}

	if values := readMultipleRegs(byte(devAddr), REG_INPUT_FREQUENCY, 4); values != nil {
		points = append(points, makePoint("IUR", int(values[1]), 0.1, 1, "R", "V", "R相输入电压"))
		points = append(points, makePoint("IUS", int(values[2]), 0.1, 1, "R", "V", "S相输入电压"))
		points = append(points, makePoint("IOT", int(values[3]), 0.1, 1, "R", "V", "T相输入电压"))
		points = append(points, makePoint("IH", int(values[0]), 0.1, 1, "R", "Hz", "输入频率"))
	}

	if values := readMultipleRegs(byte(devAddr), REG_BATTERY_CAPACITY, 2); values != nil {
		points = append(points, makePoint("qos", int(values[0]), 0.1, 1, "R", "%", "电池容量"))
		points = append(points, makePoint("ltime", int(values[1]), 1, 0, "R", "min", "电池剩余时间"))
	}

	return points
}

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
	mbap := make([]byte, 12)
	mbap[0] = 0x00
	mbap[1] = 0x01
	mbap[2] = 0x00
	mbap[3] = 0x00
	mbap[4] = 0x00
	mbap[5] = 0x06
	mbap[6] = addr

	mbap[7] = FUNC_CODE_READ
	mbap[8] = byte(startReg >> 8)
	mbap[9] = byte(startReg)
	mbap[10] = byte(count >> 8)
	mbap[11] = byte(count)

	return mbap
}

func parseReadResponse(data []byte, addr byte) ([]uint16, error) {
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

func formatFloat(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }
func errf(s string) error         { return simpleErr(s) }

func outputJSON(v interface{}) {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		b = []byte(`{"success":false,"error":"encode failed"}`)
	}
	pdk.Output(b)
}

func main() {}

