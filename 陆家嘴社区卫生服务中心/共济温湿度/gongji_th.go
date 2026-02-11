// =============================================================================
// 共济温湿度 - Modbus RTU 驱动（陆家嘴社区卫生服务中心）
// =============================================================================
//
// 协议类型: Modbus RTU
// 功能码: 0x04 (INPUT_REGISTER)
//
// 点表:
//   - 温度(temperature): 地址=0, 长度=1, 表达式=v/10
//   - 湿度(humidity): 地址=1, 长度=1, 表达式=v/10
//   - 漏点温度(dewtemperature): 地址=2, 长度=1, 表达式=v/10
//
// Host 提供: serial_transceive
//
// =============================================================================
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	pdk "github.com/extism/go-pdk"
)

//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

type DriverConfig struct {
	DeviceAddress int    `json:"device_address"`
	FuncName      string `json:"func_name"`
	FieldName     string `json:"field_name"`
	Value         string `json:"value"`
	Debug         bool   `json:"debug"`
}

const DriverVersion = "1.0.0"

const (
	REG_TEMPERATURE    = 0
	REG_HUMIDITY       = 1
	REG_DEW_TEMPERATURE = 2

	FUNC_CODE_READ_INPUT = 0x04
)

//go:wasmexport handle
func handle() int32 {
	defer func() {
		if r := recover(); r != nil {
			outputJSON(map[string]interface{}{"success": false, "error": "panic"})
		}
	}()

	cfg := getConfig()
	points := readAllPoints(cfg.DeviceAddress, cfg.Debug)

	outputJSON(map[string]interface{}{
		"success": true,
		"points":  points,
	})
	return 0
}

//go:wasmexport describe
func describe() int32 {
	outputJSON(map[string]interface{}{
		"success": true,
		"data":    map[string]string{},
	})
	return 0
}

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

func readAllPoints(devAddr int, debug bool) []map[string]interface{} {
	points := make([]map[string]interface{}, 0, 3)

	values := readMultipleRegs(byte(devAddr), REG_TEMPERATURE, 3, debug)
	if values == nil || len(values) < 3 {
		return points
	}

	points = append(points, makePoint("temperature", int64(values[0]), 0.1, 1, "R", "℃", "温度"))
	points = append(points, makePoint("humidity", int64(values[1]), 0.1, 1, "R", "%", "湿度"))
	points = append(points, makePoint("dewtemperature", int64(values[2]), 0.1, 1, "R", "℃", "漏点温度"))

	return points
}

func makePoint(field string, raw int64, scale float64, decimals int, rw, unit, label string) map[string]interface{} {
	v := float64(raw) * scale
	return map[string]interface{}{
		"field_name": field,
		"value":      formatFloat(v, decimals),
		"rw":         rw,
		"unit":       unit,
		"label":      label,
	}
}

func readMultipleRegs(devAddr byte, startReg uint16, count uint16, debug bool) []uint16 {
	req := buildReadFrame(devAddr, startReg, count)
	if debug {
		logf("rtu req=% X", req)
	}

	resp, n := serialTransceive(req, int(count)*2+5, 1000)
	if debug {
		logf("rtu n=%d resp=%s", n, hexPreview(resp, n, 16))
	}
	if n <= 0 {
		return nil
	}

	values, err := parseReadResponse(resp[:n], devAddr)
	if err != nil {
		if debug {
			logf("parse err=%v", err)
		}
		return nil
	}
	return values
}

func serialTransceive(req []byte, respLen int, timeoutMs int) ([]byte, int) {
	if len(req) == 0 || respLen <= 0 {
		return nil, 0
	}

	reqMem := pdk.AllocateBytes(req)
	defer reqMem.Free()
	respMem := pdk.Allocate(respLen)
	defer respMem.Free()

	n := int(serial_transceive(
		reqMem.Offset(), uint64(len(req)),
		respMem.Offset(), uint64(respLen),
		uint64(timeoutMs),
	))
	if n <= 0 {
		return nil, n
	}
	if n > respLen {
		n = respLen
	}

	resp := make([]byte, n)
	mem := pdk.NewMemory(respMem.Offset(), uint64(n))
	mem.Load(resp)
	return resp, n
}

func buildReadFrame(addr byte, start uint16, qty uint16) []byte {
	req := make([]byte, 8)
	req[0] = addr
	req[1] = FUNC_CODE_READ_INPUT
	req[2], req[3] = byte(start>>8), byte(start)
	req[4], req[5] = byte(qty>>8), byte(qty)
	crc := crc16(req[:6])
	req[6], req[7] = byte(crc), byte(crc>>8)
	return req
}

func parseReadResponse(data []byte, addr byte) ([]uint16, error) {
	if len(data) < 5 || data[0] != addr || data[1] != FUNC_CODE_READ_INPUT {
		return nil, errf("invalid response")
	}
	byteCnt := int(data[2])
	if byteCnt < 2 || len(data) < 3+byteCnt+2 {
		return nil, errf("byte count mismatch")
	}
	if !checkCRC(data[:3+byteCnt+2]) {
		return nil, errf("crc error")
	}

	values := make([]uint16, byteCnt/2)
	for i := 0; i < len(values); i++ {
		values[i] = uint16(data[3+i*2])<<8 | uint16(data[4+i*2])
	}
	return values, nil
}

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
	if v := strings.TrimSpace(envelope.Config["debug"]); v != "" {
		cfg.Debug = v == "1" || strings.EqualFold(v, "true")
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

func logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	pdk.Log(pdk.LogDebug, msg)
}

func hexPreview(b []byte, n int, max int) string {
	if n <= 0 {
		return ""
	}
	if n > len(b) {
		n = len(b)
	}
	if n > max {
		n = max
	}
	return fmt.Sprintf("% X", b[:n])
}

func main() {}
