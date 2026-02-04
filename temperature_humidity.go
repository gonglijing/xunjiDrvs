// =============================================================================
// 温湿度传感器 - Modbus RTU 驱动
// =============================================================================
//
// 设备点表:
//   - 温度: FC=03, 地址=0x0000, 长度=1, 缩放=0.1, 1位小数
//   - 湿度: FC=03, 地址=0x0001, 长度=1, 缩放=0.1, 1位小数
//
// Host 提供: serial_transceive
//
// =============================================================================
package main

import (
	"encoding/json"
	"strconv"
	"strings"

	pdk "github.com/extism/go-pdk"
)

// =============================================================================
// 【固定不变】Host 函数声明
// =============================================================================
//
//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

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
// 根据实际设备修改以下点表配置
const (
	// 寄存器地址定义
	// 格式: RegisterName = Address // 说明
	REG_TEMPERATURE = 0x0000 // 温度寄存器
	REG_HUMIDITY    = 0x0001 // 湿度寄存器

	// 功能码定义
	FUNC_CODE_READ  = 0x03 // 读保持寄存器
	FUNC_CODE_WRITE = 0x06 // 写单个寄存器
)

// =============================================================================
// 【用户修改】点表配置
// =============================================================================
// 定义所有需要读取的测点
// fields: 字段名, 按实际设备修改
// decimals: 有效小数位数, 按实际设备修改
var pointConfig = []PointConfig{
	{Field: "temperature", Address: REG_TEMPERATURE, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "°C", Label: "温度"},
	{Field: "humidity", Address: REG_HUMIDITY, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "%", Label: "湿度"},
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
	points := readAllPoints(cfg.DeviceAddress)

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
// 根据点表配置批量读取寄存器
func readAllPoints(devAddr int) []map[string]interface{} {
	points := make([]map[string]interface{}, 0)

	// 批量读取所有寄存器 (从第一个点表的地址开始)
	if len(pointConfig) == 0 {
		return points
	}

	// 计算需要读取的寄存器总数和起始地址
	startAddr := pointConfig[0].Address
	totalLength := uint16(0)
	for _, p := range pointConfig {
		if p.Address < startAddr {
			startAddr = p.Address
		}
		totalLength = p.Address + p.Length - startAddr
	}

	// 批量读取
	req := buildReadFrame(byte(devAddr), startAddr, totalLength)
	resp, n := serialTransceive(req, 64, 300)
	if n <= 0 {
		return points
	}

	// 解析响应
	values, err := parseReadResponse(resp[:n], byte(devAddr))
	if err != nil || len(values) < int(totalLength) {
		return points
	}

	// 将读取的值按点表配置转换为实际值
	for _, cfg := range pointConfig {
		offset := cfg.Address - startAddr
		if offset < 0 || int(offset) >= len(values) {
			continue
		}

		rawVal := values[offset]
		realVal := float64(rawVal) * cfg.Scale

		points = append(points, map[string]interface{}{
			"field_name": cfg.Field,
			"value":      formatFloat(realVal, cfg.Decimals),
			"rw":         cfg.RW,
			"unit":       cfg.Unit,
			"label":      cfg.Label,
		})
	}

	return points
}

// =============================================================================
// 【固定不变】Modbus RTU 通信函数
// =============================================================================

// 串口发送接收 (通用)
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

// 构建 Modbus RTU 读请求帧 (通用)
func buildReadFrame(addr byte, start uint16, qty uint16) []byte {
	req := make([]byte, 8)
	req[0] = addr                                // 从站地址
	req[1] = FUNC_CODE_READ                      // 功能码 0x03
	req[2], req[3] = byte(start>>8), byte(start) // 起始地址
	req[4], req[5] = byte(qty>>8), byte(qty)     // 寄存器数量
	crc := crc16(req[:6])
	req[6], req[7] = byte(crc), byte(crc>>8) // CRC 校验
	return req
}

// 解析 Modbus RTU 读响应 (通用)
func parseReadResponse(data []byte, addr byte) ([]uint16, error) {
	if len(data) < 5 || data[0] != addr || data[1] != FUNC_CODE_READ {
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

// CRC16 校验 (通用)
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

// CRC 校验 (通用)
func checkCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	got := uint16(data[len(data)-2]) | uint16(data[len(data)-1])<<8
	return crc16(data[:len(data)-2]) == got
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
