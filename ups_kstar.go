// =============================================================================
// 科士达 UPS - Modbus TCP 驱动
//   - 读取电池、输入、输出参数
//   - Host 提供 tcp_transceive、output
//   - 寄存器地址: 100~125 (int16, v/10 缩放)
//   - 优化：批量读取相邻寄存器
//
// =============================================================================
package main

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"unsafe"
)

// Host functions
//
//go:export tcp_transceive
func tcp_transceive(wPtr uint64, wSize int32, rPtr uint64, rCap int32, timeoutMs int32) int32

//go:export output
func output(ptr uint64, size int32)

// =============================================================================
// 配置结构
// =============================================================================
type UPSConfig struct {
	DeviceAddress int     `json:"device_address"` // Modbus 从站地址
	FuncName      string  `json:"func_name"`      // "read" | "write"
	FieldName     string  `json:"field_name"`     // 可写字段名
	Value         float64 `json:"value"`          // 写操作的值
}

// =============================================================================
// 驱动入口 - handle
// =============================================================================
//
//go:export handle
func handle() {
	cfg := getConfig()

	// 读操作 - 读取所有监控参数
	points := readAllUPS(cfg.DeviceAddress)

	outputJSON(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"points": points,
		},
	})
}

// =============================================================================
// 描述可写字段
// =============================================================================
//
//go:export describe
func describe() {
	writable := []map[string]interface{}{}
	outputJSON(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"writable": writable,
		},
	})
}

// =============================================================================
// 寄存器地址定义 (Modbus 协议地址)
// =============================================================================
const (
	RegBatteryCapacity   = 100 // 电池容量 (%, v/10)
	RegBatteryRemainTime = 101 // 电池剩余时间 (min)
	RegInputFrequency    = 109 // 输入频率 (Hz, v/10)
	RegInputVoltageR     = 110 // R相输入电压 (V, v/10)
	RegInputVoltageS     = 111 // S相输入电压 (V, v/10)
	RegInputVoltageT     = 112 // T相输入电压 (V, v/10)
	RegOutputFrequency   = 119 // 输出频率 (Hz, v/10)
	RegOutputVoltageR    = 120 // R相输出电压 (V, v/10)
	RegOutputVoltageS    = 121 // S相输出电压 (V, v/10)
	RegOutputVoltageT    = 122 // T相输出电压 (V, v/10)
	RegLoadPercentR      = 123 // R相负载率 (%)
	RegLoadPercentS      = 124 // S相负载率 (%)
	RegLoadPercentT      = 125 // T相负载率 (%)
)

// =============================================================================
// 读操作 - 读取所有UPS参数 (优化：批量读取)
// =============================================================================
func readAllUPS(devAddr int) []map[string]interface{} {
	points := make([]map[string]interface{}, 0)

	// ========== 电池参数 (单个读取) ==========
	// 电池容量 (v/10)
	if val := readReg(byte(devAddr), RegBatteryCapacity); val >= 0 {
		points = append(points, map[string]interface{}{
			"field_name": "battery_capacity",
			"value":      float64(val) * 0.1,
			"rw":         "R",
			"unit":       "%",
			"label":      "电池容量",
		})
	}

	// 电池剩余时间
	if val := readReg(byte(devAddr), RegBatteryRemainTime); val >= 0 {
		points = append(points, map[string]interface{}{
			"field_name": "battery_remain_time",
			"value":      float64(val),
			"rw":         "R",
			"unit":       "min",
			"label":      "电池剩余时间",
		})
	}

	// ========== 输入参数 (批量读取 109~112) ==========
	// 批量读取寄存器 109~112 (4个寄存器)
	if values := readRegs(byte(devAddr), RegInputFrequency, 4); values != nil {
		// input_frequency (v/10)
		points = append(points, map[string]interface{}{
			"field_name": "input_frequency",
			"value":      float64(values[0]) * 0.1,
			"rw":         "R",
			"unit":       "Hz",
			"label":      "输入频率",
		})

		// input_voltage_r (v/10)
		points = append(points, map[string]interface{}{
			"field_name": "input_voltage_r",
			"value":      float64(values[1]) * 0.1,
			"rw":         "R",
			"unit":       "V",
			"label":      "R相输入电压",
		})

		// input_voltage_s (v/10)
		points = append(points, map[string]interface{}{
			"field_name": "input_voltage_s",
			"value":      float64(values[2]) * 0.1,
			"rw":         "R",
			"unit":       "V",
			"label":      "S相输入电压",
		})

		// input_voltage_t (v/10)
		points = append(points, map[string]interface{}{
			"field_name": "input_voltage_t",
			"value":      float64(values[3]) * 0.1,
			"rw":         "R",
			"unit":       "V",
			"label":      "T相输入电压",
		})
	}

	// ========== 输出参数 (批量读取 119~125) ==========
	// 批量读取寄存器 119~125 (7个寄存器)
	if values := readRegs(byte(devAddr), RegOutputFrequency, 7); values != nil {
		// output_frequency (v/10)
		points = append(points, map[string]interface{}{
			"field_name": "output_frequency",
			"value":      float64(values[0]) * 0.1,
			"rw":         "R",
			"unit":       "Hz",
			"label":      "输出频率",
		})

		// output_voltage_r (v/10)
		points = append(points, map[string]interface{}{
			"field_name": "output_voltage_r",
			"value":      float64(values[1]) * 0.1,
			"rw":         "R",
			"unit":       "V",
			"label":      "R相输出电压",
		})

		// output_voltage_s (v/10)
		points = append(points, map[string]interface{}{
			"field_name": "output_voltage_s",
			"value":      float64(values[2]) * 0.1,
			"rw":         "R",
			"unit":       "V",
			"label":      "S相输出电压",
		})

		// output_voltage_t (v/10)
		points = append(points, map[string]interface{}{
			"field_name": "output_voltage_t",
			"value":      float64(values[3]) * 0.1,
			"rw":         "R",
			"unit":       "V",
			"label":      "T相输出电压",
		})

		// load_percent_r
		points = append(points, map[string]interface{}{
			"field_name": "load_percent_r",
			"value":      float64(values[4]),
			"rw":         "R",
			"unit":       "%",
			"label":      "R相负载率",
		})

		// load_percent_s
		points = append(points, map[string]interface{}{
			"field_name": "load_percent_s",
			"value":      float64(values[5]),
			"rw":         "R",
			"unit":       "%",
			"label":      "S相负载率",
		})

		// load_percent_t
		points = append(points, map[string]interface{}{
			"field_name": "load_percent_t",
			"value":      float64(values[6]),
			"rw":         "R",
			"unit":       "%",
			"label":      "T相负载率",
		})
	}

	return points
}

// =============================================================================
// Modbus 通信函数
// =============================================================================

// 读取单个寄存器 (0x03)
func readReg(devAddr byte, regAddr uint16) int {
	values := readRegs(devAddr, regAddr, 1)
	if values == nil || len(values) < 1 {
		return -1
	}
	return int(values[0])
}

// 批量读取寄存器 (0x03)
func readRegs(devAddr byte, startReg uint16, count uint16) []int16 {
	req := buildModbusRequest(devAddr, 0x03, startReg, count)
	resp := make([]byte, 64)

	n := tcp_transceive(
		uint64(uintptr(unsafe.Pointer(&req[0]))), int32(len(req)),
		uint64(uintptr(unsafe.Pointer(&resp[0]))), int32(len(resp)),
		1000,
	)

	if n < 7 {
		return nil
	}

	// 解析响应
	values, err := parseReadResponse(resp[:n], devAddr, 0x03)
	if err != nil || len(values) < int(count) {
		return nil
	}

	// 转换为 int16
	result := make([]int16, count)
	for i := 0; i < int(count); i++ {
		result[i] = int16(values[i])
	}

	return result
}

// 构建 Modbus TCP 请求帧
func buildModbusRequest(addr byte, funcCode byte, startReg uint16, count uint16) []byte {
	// MBAP 头 (7字节) + PDU (5字节) = 12字节
	mbap := make([]byte, 12)
	mbap[0] = 0x00 // 事务标识符高字节
	mbap[1] = 0x01 // 事务标识符低字节
	mbap[2] = 0x00 // 协议标识符高字节
	mbap[3] = 0x00 // 协议标识符低字节
	mbap[4] = 0x00 // 长度高字节
	mbap[5] = 0x06 // 长度低字节 (6字节: 地址 + 功能码 + 寄存器地址 + 数量)
	mbap[6] = addr // 单元标识符

	// PDU
	mbap[7] = funcCode            // 功能码 (0x03 读保持寄存器)
	mbap[8] = byte(startReg >> 8) // 寄存器地址高字节
	mbap[9] = byte(startReg)      // 寄存器地址低字节
	mbap[10] = byte(count >> 8)   // 数量高字节
	mbap[11] = byte(count)        // 数量低字节

	return mbap
}

// 解析读响应
func parseReadResponse(data []byte, addr byte, funcCode byte) ([]uint16, error) {
	// 跳过 MBAP 头 (7字节)
	pdu := data[6:]

	if len(pdu) < 3 {
		return nil, errf("响应数据不完整")
	}

	if pdu[0] != addr || pdu[1] != funcCode {
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
// 工具函数
// =============================================================================

// 获取配置
func getConfig() UPSConfig {
	def := UPSConfig{DeviceAddress: 1, FuncName: "read"}

	//go:extern extism_input_ptr
	var extism_input_ptr *byte
	//go:extern extism_input_len
	var extism_input_len uint32
	if extism_input_ptr == nil || extism_input_len == 0 {
		return def
	}

	hdr := &reflect.SliceHeader{Data: uintptr(unsafe.Pointer(extism_input_ptr)), Len: int(extism_input_len), Cap: int(extism_input_len)}
	data := *(*[]byte)(unsafe.Pointer(hdr))

	var envelope struct {
		Config UPSConfig `json:"config"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return def
	}

	cfg := envelope.Config
	if cfg.DeviceAddress == 0 {
		cfg.DeviceAddress = def.DeviceAddress
	}
	if cfg.FuncName == "" {
		cfg.FuncName = def.FuncName
	}
	return cfg
}

// 输出 JSON
func outputJSON(v interface{}) {
	b, _ := json.Marshal(v)
	ptr := uint64(uintptr(unsafe.Pointer(unsafe.StringData(string(b)))))
	output(ptr, int32(len(b)))
}

// 轻量错误
type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errf(s string) error { return simpleErr(s) }

func main() {}
