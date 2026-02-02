// =============================================================================
// 温湿度传感器 - Modbus TCP 示例
//   - 读取温度/湿度 (0x0000~0x0001)
//   - 读取温度告警阈值 (0x0022)
//   - 写设备地址 (0x0021) / 写温度阈值 (0x0022)
//   - Host 提供 tcp_transceive、output
//
// =============================================================================
package main

import (
	"encoding/json"
	"reflect"
	"unsafe"
)

// Host functions
//
//go:export tcp_transceive
func tcp_transceive(wPtr unsafe.Pointer, wSize, rPtr, rCap, timeoutMs int32) int32

//go:export output
func output(ptr unsafe.Pointer, size int32)

type GatewayConfig struct {
	DeviceAddress int     `json:"device_address"`
	FuncName      string  `json:"func_name"`  // "read" | "write"
	FieldName     string  `json:"field_name"` // device_addr | temp_alarm_threshold
	Value         float64 `json:"value"`
}

//go:export handle
func handle() {
	cfg := getConfig()
	if cfg.FuncName == "write" {
		if !doWrite(cfg) {
			outputJSON(map[string]interface{}{"success": false, "error": "write failed"})
			return
		}
		outputJSON(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"field": cfg.FieldName,
				"value": cfg.Value,
			},
		})
		return
	}

	points, ok := readAll(cfg.DeviceAddress)
	if !ok {
		outputJSON(map[string]interface{}{"success": false, "error": "read or decode failed"})
		return
	}
	outputJSON(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"points": points,
		},
	})
}

//go:export describe
func describe() {
	writable := []map[string]interface{}{
		{"field": "device_addr", "label": "设备地址", "desc": "写入新的设备地址"},
		{"field": "temp_alarm_threshold", "label": "温度告警阈值", "desc": "设置温度告警阈值"},
	}
	outputJSON(map[string]interface{}{
		"success": true,
		"data":    map[string]interface{}{"writable": writable},
	})
}

// 读取温湿度 + 阈值
func readAll(devAddr int) ([]map[string]interface{}, bool) {
	points := make([]map[string]interface{}, 0)

	// 温湿度
	req := buildReadFrameTCP(byte(devAddr), 0x0000, 0x0002)
	resp := make([]byte, 64)
	if n := tcpTransceive(req, resp, 300); n > 0 {
		if ps, err := decodeMultiRespTCP(resp[:n]); err == nil {
			points = append(points, ps...)
		}
	}

	// 阈值
	req2 := buildReadFrameTCP(byte(devAddr), 0x0022, 0x0001)
	resp2 := make([]byte, 32)
	if n2 := tcpTransceive(req2, resp2, 300); n2 > 0 {
		if th, err := decodeSingleTCP(resp2[:n2]); err == nil {
			points = append(points, map[string]interface{}{
				"field_name": "temp_alarm_threshold",
				"value":      th,
				"rw":         "RW",
			})
		}
	}

	return points, len(points) > 0
}

func doWrite(cfg GatewayConfig) bool {
	switch cfg.FieldName {
	case "device_addr", "":
		return writeRegisterTCP(cfg.DeviceAddress, 0x0021, uint16(cfg.Value))
	case "temp_alarm_threshold":
		return writeRegisterTCP(cfg.DeviceAddress, 0x0022, uint16(cfg.Value))
	default:
		return false
	}
}

// TCP 0x03 读
func buildReadFrameTCP(unit byte, start uint16, qty uint16) []byte {
	req := make([]byte, 12)
	req[4], req[5] = 0x00, 0x06
	req[6] = unit
	req[7] = 0x03
	req[8], req[9] = byte(start>>8), byte(start)
	req[10], req[11] = byte(qty>>8), byte(qty)
	return req
}

// TCP 0x06 写
func buildWriteFrameTCP(unit byte, reg uint16, val uint16) []byte {
	req := make([]byte, 12)
	req[4], req[5] = 0x00, 0x06
	req[6] = unit
	req[7] = 0x06
	req[8], req[9] = byte(reg>>8), byte(reg)
	req[10], req[11] = byte(val>>8), byte(val)
	return req
}

func tcpTransceive(req []byte, resp []byte, timeoutMs int) int {
	if len(req) == 0 || len(resp) == 0 {
		return 0
	}
	return int(tcp_transceive(
		unsafe.Pointer(&req[0]), int32(len(req)),
		unsafe.Pointer(&resp[0]), int32(len(resp)),
		int32(timeoutMs),
	))
}

// 解码温湿度
func decodeMultiRespTCP(resp []byte) ([]map[string]interface{}, error) {
	if len(resp) < 11 || resp[7] != 0x03 {
		return nil, errf("invalid resp")
	}
	byteCnt := int(resp[8])
	if byteCnt < 4 || len(resp) < 9+byteCnt {
		return nil, errf("byte count mismatch")
	}
	data := resp[9 : 9+byteCnt]
	tempRaw := int16(data[0])<<8 | int16(data[1])
	humiRaw := int16(data[2])<<8 | int16(data[3])
	points := []map[string]interface{}{
		{"field_name": "temperature", "value": float64(tempRaw) * 0.1, "rw": "R"},
		{"field_name": "humidity", "value": float64(humiRaw) * 0.1, "rw": "R"},
	}
	return points, nil
}

// 解码单寄存
func decodeSingleTCP(resp []byte) (float64, error) {
	if len(resp) < 11 || resp[7] != 0x03 || resp[8] < 2 {
		return 0, errf("invalid resp")
	}
	val := int16(resp[9])<<8 | int16(resp[10])
	return float64(val), nil
}

func writeRegisterTCP(devAddr int, reg uint16, val uint16) bool {
	req := buildWriteFrameTCP(byte(devAddr), reg, val)
	resp := make([]byte, 24)
	if n := tcpTransceive(req, resp, 300); n < 12 {
		return false
	}
	return resp[7] == 0x06
}

// getConfig：从 Extism 输入的 envelope.config 解析
func getConfig() GatewayConfig {
	def := GatewayConfig{DeviceAddress: 1, FuncName: "read"}

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
		Config GatewayConfig `json:"config"`
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

// 轻量错误
type simpleErr string

func (e simpleErr) Error() string { return string(e) }
func errf(s string) error         { return simpleErr(s) }

func outputJSON(v interface{}) {
	b, _ := json.Marshal(v)
	ptr := unsafe.Pointer(unsafe.StringData(string(b)))
	output(ptr, int32(len(b)))
}

func main() {}
