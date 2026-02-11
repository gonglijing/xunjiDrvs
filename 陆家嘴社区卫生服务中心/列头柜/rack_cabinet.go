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
	"encoding/binary"
	"encoding/json"
	"strconv"
	"strings"

	pdk "github.com/extism/go-pdk"
)

//go:wasmimport extism:host/user tcp_transceive
func tcp_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

type DriverConfig struct {
	DeviceAddress int    `json:"device_address"`
	FuncName      string `json:"func_name"`
	FieldName     string `json:"field_name"`
	Value         string `json:"value"`
}

const DriverVersion = "1.0.0"

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
)

//go:wasmexport handle
func handle() int32 {
	defer func() {
		if r := recover(); r != nil {
			outputJSON(map[string]interface{}{"success": false, "error": "panic"})
		}
	}()

	cfg := getConfig()
	points := readAllPoints(cfg.DeviceAddress)

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

func readAllPoints(devAddr int) []map[string]interface{} {
	points := make([]map[string]interface{}, 0, 80)

	if values := readMultipleRegs(byte(devAddr), REG_VOLTAGE_START, REG_VOLTAGE_LEN); values != nil {
		points = append(points, makeScaledPoint("UA1", int64(values[0]), 0.1, 1, "R", "V", "市电总输入A"))
		points = append(points, makeScaledPoint("UB1", int64(values[1]), 0.1, 1, "R", "V", "市电总输入B"))
		points = append(points, makeScaledPoint("UC1", int64(values[2]), 0.1, 1, "R", "V", "市电总输入C"))
		points = append(points, makeScaledPoint("Uups", int64(values[3]), 0.1, 1, "R", "V", "UPS输出"))
	}

	if values := readMultipleRegs(byte(devAddr), REG_CURRENT_START, REG_CURRENT_LEN); values != nil {
		points = append(points, makeScaledPoint("MainsACurr", int64(values[0]), 0.1, 1, "R", "A", "市电输入A相电流"))
		points = append(points, makeScaledPoint("MainsBCurr", int64(values[1]), 0.1, 1, "R", "A", "市电输入B相电流"))
		points = append(points, makeScaledPoint("MainsCCurr", int64(values[2]), 0.1, 1, "R", "A", "市电输入C相电流"))
		points = append(points, makeScaledPoint("UPSIC", int64(values[3]), 0.1, 1, "R", "A", "UPS输入总电流"))
		points = append(points, makeScaledPoint("UPSACurr", int64(values[4]), 0.1, 1, "R", "A", "UPS输出A相电流"))
		points = append(points, makeScaledPoint("UPSBCurr", int64(values[5]), 0.1, 1, "R", "A", "UPS输出B相电流"))
		points = append(points, makeScaledPoint("UPSCCurr", int64(values[6]), 0.1, 1, "R", "A", "UPS输出C相电流"))
		points = append(points, makeScaledPoint("MainsPdu1Curr", int64(values[7]), 0.1, 1, "R", "A", "市电PDU-1电流"))
		points = append(points, makeScaledPoint("MainsPdu2Curr", int64(values[8]), 0.1, 1, "R", "A", "市电PDU-2电流"))
		points = append(points, makeScaledPoint("MainsPdu3Curr", int64(values[9]), 0.1, 1, "R", "A", "市电PDU-3电流"))
		points = append(points, makeScaledPoint("MainsPdu4Curr", int64(values[10]), 0.1, 1, "R", "A", "市电PDU-4电流"))
		points = append(points, makeScaledPoint("MainsPdu5Curr", int64(values[11]), 0.1, 1, "R", "A", "市电PDU-5电流"))
		points = append(points, makeScaledPoint("MainsPdu6Curr", int64(values[12]), 0.1, 1, "R", "A", "市电PDU-6电流"))
		points = append(points, makeScaledPoint("MainsPdu7Curr", int64(values[13]), 0.1, 1, "R", "A", "市电PDU-7电流"))
		points = append(points, makeScaledPoint("UpsPdu1Curr", int64(values[14]), 0.1, 1, "R", "A", "U电PDU-1电流"))
		points = append(points, makeScaledPoint("UpsPdu2Curr", int64(values[15]), 0.1, 1, "R", "A", "U电PDU-2电流"))
		points = append(points, makeScaledPoint("UpsPdu3Curr", int64(values[16]), 0.1, 1, "R", "A", "U电PDU-3电流"))
		points = append(points, makeScaledPoint("UpsPdu4Curr", int64(values[17]), 0.1, 1, "R", "A", "U电PDU-4电流"))
		points = append(points, makeScaledPoint("UpsPdu5Curr", int64(values[18]), 0.1, 1, "R", "A", "U电PDU-5电流"))
		points = append(points, makeScaledPoint("UpsPdu6Curr", int64(values[19]), 0.1, 1, "R", "A", "U电PDU-6电流"))
		points = append(points, makeScaledPoint("UpsPdu7Curr", int64(values[20]), 0.1, 1, "R", "A", "U电PDU-7电流"))
	}

	if values := readMultipleRegs(byte(devAddr), REG_POWER_START, REG_POWER_LEN); values != nil {
		points = append(points, makeScaledPoint("MainsPA", int64(values[0]), 0.1, 1, "R", "kW", "市电输出A相功率"))
		points = append(points, makeScaledPoint("MainsPB", int64(values[1]), 0.1, 1, "R", "kW", "市电输出B相功率"))
		points = append(points, makeScaledPoint("MainsPC", int64(values[2]), 0.1, 1, "R", "kW", "市电输出C相功率"))
		points = append(points, makeScaledPoint("MainsPdu1P", int64(values[3]), 0.1, 1, "R", "kW", "市电PDU1功率"))
		points = append(points, makeScaledPoint("MainsPdu2P", int64(values[4]), 0.1, 1, "R", "kW", "市电PDU2功率"))
		points = append(points, makeScaledPoint("MainsPdu3P", int64(values[5]), 0.1, 1, "R", "kW", "市电PDU3功率"))
		points = append(points, makeScaledPoint("MainsPdu4P", int64(values[6]), 0.1, 1, "R", "kW", "市电PDU4功率"))
		points = append(points, makeScaledPoint("MainsPdu5P", int64(values[7]), 0.1, 1, "R", "kW", "市电PDU5功率"))
		points = append(points, makeScaledPoint("MainsPdu6P", int64(values[8]), 0.1, 1, "R", "kW", "市电PDU6功率"))
		points = append(points, makeScaledPoint("MainsPdu7P", int64(values[9]), 0.1, 1, "R", "kW", "市电PDU7功率"))
		points = append(points, makeScaledPoint("UpsPdu1P", int64(values[10]), 0.1, 1, "R", "kW", "U电PDU1功率"))
		points = append(points, makeScaledPoint("UpsPdu2P", int64(values[11]), 0.1, 1, "R", "kW", "U电PDU2功率"))
		points = append(points, makeScaledPoint("UpsPdu3P", int64(values[12]), 0.1, 1, "R", "kW", "U电PDU3功率"))
		points = append(points, makeScaledPoint("UpsPdu4P", int64(values[13]), 0.1, 1, "R", "kW", "U电PDU4功率"))
		points = append(points, makeScaledPoint("UpsPdu5P", int64(values[14]), 0.1, 1, "R", "kW", "U电PDU5功率"))
		points = append(points, makeScaledPoint("UpsPdu6P", int64(values[15]), 0.1, 1, "R", "kW", "U电PDU6功率"))
		points = append(points, makeScaledPoint("UpsPdu7P", int64(values[16]), 0.1, 1, "R", "kW", "U电PDU7功率"))
	}

	if values := readMultipleRegs(byte(devAddr), REG_ENERGY_START, REG_ENERGY_LEN); values != nil {
		if raw, ok := readU32(values, REG_ENERGY_START, 854); ok {
			points = append(points, makeScaledPoint("MainsEPA", raw, 0.1, 1, "R", "kWh", "市电输出A相电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 856); ok {
			points = append(points, makeScaledPoint("MainsEPB", raw, 0.1, 1, "R", "kWh", "市电输出B相电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 858); ok {
			points = append(points, makeScaledPoint("MainsEPC", raw, 0.1, 1, "R", "kWh", "市电输出C相电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 860); ok {
			points = append(points, makeScaledPoint("MainsPdu1EP", raw, 0.1, 1, "R", "kWh", "市电PDU1电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 848); ok {
			points = append(points, makeScaledPoint("MainsPdu2EP", raw, 0.1, 1, "R", "kWh", "市电PDU2电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 850); ok {
			points = append(points, makeScaledPoint("MainsPdu3EP", raw, 0.1, 1, "R", "kWh", "市电PDU3电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 866); ok {
			points = append(points, makeScaledPoint("MainsPdu4EP", raw, 0.1, 1, "R", "kWh", "市电PDU4电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 868); ok {
			points = append(points, makeScaledPoint("MainsPdu5EP", raw, 0.1, 1, "R", "kWh", "市电PDU5电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 870); ok {
			points = append(points, makeScaledPoint("MainsPdu6EP", raw, 0.1, 1, "R", "kWh", "市电PDU6电能"))
		}
		if raw, ok := readU32(values, REG_ENERGY_START, 872); ok {
			points = append(points, makeScaledPoint("MainsPdu7EP", raw, 0.1, 1, "R", "kWh", "市电PDU7电能"))
		}
	}

	if values := readMultipleRegs(byte(devAddr), REG_SWITCH_START, REG_SWITCH_LEN); values != nil {
		points = append(points, makeSwitchPoint("MSS", values[0], "市电总输入开关状态"))
		points = append(points, makeSwitchPoint("MainsPdu1Switch", values[3], "市电PDU1开关状态"))
		points = append(points, makeSwitchPoint("MainsPdu2Switch", values[4], "市电PDU2开关状态"))
		points = append(points, makeSwitchPoint("MainsPdu3Switch", values[5], "市电PDU3开关状态"))
		points = append(points, makeSwitchPoint("MainsPdu4Switch", values[6], "市电PDU4开关状态"))
		points = append(points, makeSwitchPoint("MainsPdu5Switch", values[7], "市电PDU5开关状态"))
		points = append(points, makeSwitchPoint("MainsPdu6Switch", values[8], "市电PDU6开关状态"))
		points = append(points, makeSwitchPoint("MainsPdu7Switch", values[9], "市电PDU7开关状态"))
		points = append(points, makeSwitchPoint("UpsPdu1Switch", values[10], "U电PDU1开关状态"))
		points = append(points, makeSwitchPoint("UpsPdu2Switch", values[11], "U电PDU2开关状态"))
		points = append(points, makeSwitchPoint("UpsPdu3Switch", values[12], "U电PDU3开关状态"))
		points = append(points, makeSwitchPoint("UpsPdu4Switch", values[13], "U电PDU4开关状态"))
		points = append(points, makeSwitchPoint("UpsPdu5Switch", values[14], "U电PDU5开关状态"))
		points = append(points, makeSwitchPoint("UpsPdu6Switch", values[15], "U电PDU6开关状态"))
		points = append(points, makeSwitchPoint("UpsPdu7Switch", values[16], "U电PDU7开关状态"))
	}

	return points
}

func makeScaledPoint(field string, raw int64, scale float64, decimals int, rw, unit, label string) map[string]interface{} {
	realVal := float64(raw) * scale
	return map[string]interface{}{
		"field_name": field,
		"value":      formatFloat(realVal, decimals),
		"rw":         rw,
		"unit":       unit,
		"label":      label,
	}
}

func makeSwitchPoint(field string, raw uint16, label string) map[string]interface{} {
	v := int64(raw & 0x8000)
	return map[string]interface{}{
		"field_name": field,
		"value":      strconv.FormatInt(v, 10),
		"rw":         "R",
		"unit":       "",
		"label":      label,
	}
}

func readU32(values []uint16, startReg uint16, targetReg uint16) (int64, bool) {
	idx := int(targetReg - startReg)
	if idx < 0 || idx+1 >= len(values) {
		return 0, false
	}
	v := binary.BigEndian.Uint32([]byte{
		byte(values[idx] >> 8), byte(values[idx]),
		byte(values[idx+1] >> 8), byte(values[idx+1]),
	})
	return int64(v), true
}

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
		start := 3 + i*2
		values[i] = binary.BigEndian.Uint16(pdu[start : start+2])
	}
	return values, nil
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
