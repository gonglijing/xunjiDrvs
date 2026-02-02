// =============================================================================
// 温湿度传感器 - Modbus RTU 示例
//   - 读取温度/湿度 (0x0000~0x0001)
//   - 读取温度告警阈值 (0x0022)
//   - 写设备地址 (0x0021) / 写温度阈值 (0x0022)
//   - Host 提供 serial_transceive、output
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
//go:export serial_transceive
func serial_transceive(wPtr unsafe.Pointer, wSize, rPtr, rCap, timeoutMs int32) int32

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

// 读取温度/湿度 + 阈值
func readAll(devAddr int) ([]map[string]interface{}, bool) {
	points := make([]map[string]interface{}, 0)

	// 温湿度 (0x0000~0x0001)
	req := buildReadFrame(byte(devAddr), 0x0000, 0x0002)
	resp := make([]byte, 32)
	n := serial_transceive(
		unsafe.Pointer(&req[0]), int32(len(req)),
		unsafe.Pointer(&resp[0]), int32(len(resp)),
		300,
	)
	if ps, err := decodeMulti(resp[:n], byte(devAddr)); err == nil {
		points = append(points, ps...)
	}

	// 温度阈值 (0x0022)
	req2 := buildReadFrame(byte(devAddr), 0x0022, 0x0001)
	resp2 := make([]byte, 16)
	n2 := serial_transceive(
		unsafe.Pointer(&req2[0]), int32(len(req2)),
		unsafe.Pointer(&resp2[0]), int32(len(resp2)),
		300,
	)
	if th, err := decodeSingle(resp2[:n2], byte(devAddr)); err == nil {
		points = append(points, map[string]interface{}{
			"field_name": "temp_alarm_threshold",
			"value":      th,
			"rw":         "RW",
		})
	}

	return points, len(points) > 0
}

func doWrite(cfg GatewayConfig) bool {
	switch cfg.FieldName {
	case "device_addr", "":
		return writeRegister(cfg.DeviceAddress, 0x0021, uint16(cfg.Value))
	case "temp_alarm_threshold":
		return writeRegister(cfg.DeviceAddress, 0x0022, uint16(cfg.Value))
	default:
		return false
	}
}

// 0x03 读帧
func buildReadFrame(addr byte, start uint16, qty uint16) []byte {
	req := make([]byte, 8)
	req[0] = addr
	req[1] = 0x03
	req[2], req[3] = byte(start>>8), byte(start)
	req[4], req[5] = byte(qty>>8), byte(qty)
	crc := crc16(req[:6])
	req[6], req[7] = byte(crc), byte(crc>>8)
	return req
}

// 0x06 写帧
func buildWriteFrame(addr byte, reg uint16, val uint16) []byte {
	req := make([]byte, 8)
	req[0] = addr
	req[1] = 0x06
	req[2], req[3] = byte(reg>>8), byte(reg)
	req[4], req[5] = byte(val>>8), byte(val)
	crc := crc16(req[:6])
	req[6], req[7] = byte(crc), byte(crc>>8)
	return req
}

func writeRegister(devAddr int, reg uint16, val uint16) bool {
	req := buildWriteFrame(byte(devAddr), reg, val)
	resp := make([]byte, 16)
	n := serial_transceive(
		unsafe.Pointer(&req[0]), int32(len(req)),
		unsafe.Pointer(&resp[0]), int32(len(resp)),
		300,
	)
	if n < 8 || !checkCRC(resp[:n]) {
		return false
	}
	return resp[0] == byte(devAddr) && resp[1] == 0x06
}

// 解析温湿度
func decodeMulti(resp []byte, dev byte) ([]map[string]interface{}, error) {
	if len(resp) < 9 || resp[0] != dev || resp[1] != 0x03 {
		return nil, errf("invalid resp")
	}
	byteCnt := int(resp[2])
	if byteCnt < 4 || len(resp) < 3+byteCnt+2 {
		return nil, errf("byte count mismatch")
	}
	if !checkCRC(resp[:3+byteCnt]) {
		return nil, errf("crc error")
	}
	data := resp[3 : 3+byteCnt]
	tempRaw := int16(data[0])<<8 | int16(data[1])
	humiRaw := int16(data[2])<<8 | int16(data[3])
	points := []map[string]interface{}{
		{"field_name": "temperature", "value": float64(tempRaw) * 0.1, "rw": "R"},
		{"field_name": "humidity", "value": float64(humiRaw) * 0.1, "rw": "R"},
	}
	return points, nil
}

// 解析单寄存
func decodeSingle(resp []byte, dev byte) (float64, error) {
	if len(resp) < 7 || resp[0] != dev || resp[1] != 0x03 || resp[2] < 2 {
		return 0, errf("invalid resp")
	}
	if !checkCRC(resp[:len(resp)-2]) {
		return 0, errf("crc error")
	}
	val := int16(resp[3])<<8 | int16(resp[4])
	return float64(val), nil
}

// CRC16
func crc16(data []byte) uint16 {
	var crc uint16 = 0xFFFF
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func checkCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	got := uint16(data[len(data)-2]) | uint16(data[len(data)-1])<<8
	return crc16(data[:len(data)-2]) == got
}

// 获取配置：从 Extism 输入 envelope.config 解析
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
