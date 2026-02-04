// =============================================================================
// 温湿度传感器 - Modbus TCP 驱动
// =============================================================================
//
// 设备点表:
//   - 温度: FC=03, 地址=0x0000, 长度=1, 缩放=0.1, 1位小数
//   - 湿度: FC=03, 地址=0x0001, 长度=1, 缩放=0.1, 1位小数
//   - 温度告警阈值: FC=03, 地址=0x0022, 长度=1, 缩放=1, 0位小数, RW
//
// Host 提供: tcp_transceive
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
// 根据实际设备修改以下点表配置
const (
	// 寄存器地址定义
	// 格式: RegisterName = Address // 说明
	REG_TEMPERATURE          = 0x0000 // 温度寄存器
	REG_HUMIDITY             = 0x0001 // 湿度寄存器
	REG_TEMP_ALARM_THRESHOLD = 0x0022 // 温度告警阈值寄存器

	// 功能码定义
	FUNC_CODE_READ  = 0x03 // 读保持寄存器
	FUNC_CODE_WRITE = 0x06 // 写单个寄存器
)

// =============================================================================
// 【用户修改】点表配置
// =============================================================================
// 定义所有需要读取的测点
var pointConfig = []PointConfig{
	{Field: "temperature", Address: REG_TEMPERATURE, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "°C", Label: "温度"},
	{Field: "humidity", Address: REG_HUMIDITY, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "%", Label: "湿度"},
	{Field: "temp_alarm_threshold", Address: REG_TEMP_ALARM_THRESHOLD, Length: 1, Scale: 1, Decimals: 0, RW: "RW", Unit: "°C", Label: "温度告警阈值"},
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

	// 写操作
	if cfg.FuncName == "write" {
		if !doWrite(cfg) {
			outputJSON(map[string]interface{}{"success": false, "error": "write failed"})
			return 0
		}
		outputJSON(map[string]interface{}{
			"success": true,
			"points": []map[string]interface{}{
				{
					"field_name": cfg.FieldName,
					"value":      cfg.Value,
					"rw":         "W",
				},
			},
		})
		return 0
	}

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
	maxEndAddr := uint16(0)
	for _, p := range pointConfig {
		if p.Address < startAddr {
			startAddr = p.Address
		}
		endAddr := p.Address + p.Length
		if endAddr > maxEndAddr {
			maxEndAddr = endAddr
		}
	}
	totalLength := maxEndAddr - startAddr

	// 批量读取
	req := buildReadRequest(byte(devAddr), startAddr, totalLength)
	resp := make([]byte, 64)
	n := tcpTransceive(req, resp, 300)
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
// 【用户修改】写操作
// =============================================================================
// 根据实际设备修改写逻辑
func doWrite(cfg DriverConfig) bool {
	switch cfg.FieldName {
	case "device_addr", "":
		// 写设备地址
		val, err := strconv.ParseFloat(cfg.Value, 64)
		if err != nil {
			return false
		}
		return writeRegister(byte(cfg.DeviceAddress), REG_TEMP_ALARM_THRESHOLD-1, uint16(val))
	case "temp_alarm_threshold":
		// 写温度告警阈值
		val, err := strconv.ParseFloat(cfg.Value, 64)
		if err != nil {
			return false
		}
		return writeRegister(byte(cfg.DeviceAddress), REG_TEMP_ALARM_THRESHOLD, uint16(val))
	default:
		return false
	}
}

// =============================================================================
// 【固定不变】Modbus TCP 通信函数
// =============================================================================

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
func buildReadRequest(unit byte, startReg uint16, count uint16) []byte {
	// MBAP 头 (7字节) + PDU (5字节) = 12字节
	mbap := make([]byte, 12)
	mbap[0] = 0x00 // 事务标识符
	mbap[1] = 0x01
	mbap[2] = 0x00 // 协议标识符
	mbap[3] = 0x00
	mbap[4] = 0x00 // 长度高字节
	mbap[5] = 0x06 // 长度低字节 (6字节)
	mbap[6] = unit // 单元标识符 (从站地址)

	// PDU
	mbap[7] = FUNC_CODE_READ      // 功能码
	mbap[8] = byte(startReg >> 8) // 起始地址高字节
	mbap[9] = byte(startReg)      // 起始地址低字节
	mbap[10] = byte(count >> 8)   // 数量高字节
	mbap[11] = byte(count)        // 数量低字节

	return mbap
}

// 构建 Modbus TCP 写请求帧 (通用)
func buildWriteRequest(unit byte, reg uint16, val uint16) []byte {
	mbap := make([]byte, 12)
	mbap[0] = 0x00
	mbap[1] = 0x01
	mbap[2] = 0x00
	mbap[3] = 0x00
	mbap[4] = 0x00
	mbap[5] = 0x06
	mbap[6] = unit

	mbap[7] = FUNC_CODE_WRITE // 功能码 0x06
	mbap[8] = byte(reg >> 8)  // 寄存器地址高字节
	mbap[9] = byte(reg)       // 寄存器地址低字节
	mbap[10] = byte(val >> 8) // 值高字节
	mbap[11] = byte(val)      // 值低字节

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
		values[i] = uint16(pdu[3+i*2])<<8 | uint16(pdu[4+i*2])
	}

	return values, nil
}

// 写单个寄存器 (通用)
func writeRegister(unit byte, reg uint16, val uint16) bool {
	req := buildWriteRequest(unit, reg, val)
	resp := make([]byte, 24)
	if n := tcpTransceive(req, resp, 300); n < 12 {
		return false
	}
	return resp[7] == FUNC_CODE_WRITE
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
