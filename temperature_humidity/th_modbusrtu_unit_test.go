// =============================================================================
// 温湿度传感器 - Modbus RTU 驱动 单元测试
// =============================================================================
//
// 测试策略：
// - 创建独立的核心函数副本进行测试
// - 不依赖 WebAssembly/PDK 环境
// - 可以直接使用 go test 运行
//
// =============================================================================
package main

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
)

// =============================================================================
// 核心函数副本 (从 th_modbusrtu.go 复制，不依赖 PDK)
// =============================================================================

// CRC16 校验 (Modbus RTU 标准算法)
func crc16_local(data []byte) uint16 {
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

// CRC 校验
func checkCRC_local(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	got := uint16(data[len(data)-2]) | uint16(data[len(data)-1])<<8
	return crc16_local(data[:len(data)-2]) == got
}

// 构建 Modbus RTU 读请求帧
func buildReadFrame_local(addr byte, start uint16, qty uint16) []byte {
	req := make([]byte, 8)
	req[0] = addr                                // 从站地址
	req[1] = 0x03                                // 功能码 0x03
	req[2], req[3] = byte(start>>8), byte(start) // 起始地址
	req[4], req[5] = byte(qty>>8), byte(qty)     // 寄存器数量
	crc := crc16_local(req[:6])
	req[6], req[7] = byte(crc), byte(crc>>8) // CRC 校验
	return req
}

// 解析 Modbus RTU 读响应
func parseReadResponse_local(data []byte, addr byte) ([]uint16, error) {
	if len(data) < 5 || data[0] != addr || data[1] != 0x03 {
		return nil, fmt.Errorf("invalid response")
	}
	byteCnt := int(data[2])
	if byteCnt < 2 || len(data) < 3+byteCnt+2 {
		return nil, fmt.Errorf("byte count mismatch")
	}
	if !checkCRC_local(data[:3+byteCnt+2]) {
		return nil, fmt.Errorf("crc error")
	}

	values := make([]uint16, byteCnt/2)
	for i := 0; i < len(values); i++ {
		values[i] = uint16(data[3+i*2])<<8 | uint16(data[4+i*2])
	}
	return values, nil
}

// 格式化浮点数
func formatFloat_local(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

// 错误类型
type simpleErr_local string

func (e simpleErr_local) Error() string { return string(e) }

func errf_local(s string) error { return simpleErr_local(s) }

// 十六进制预览
func hexPreview_local(b []byte, n int, max int) string {
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

// 转义字符串
func appendEscaped_local(dst []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '"':
			dst = append(dst, '\\', s[i])
		default:
			dst = append(dst, s[i])
		}
	}
	return dst
}

// =============================================================================
// Test: CRC16 校验函数
// =============================================================================

func TestCRC16_Consistency(t *testing.T) {
	// 验证 CRC16 的一致性（同输入同输出）
	input := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x02}

	result1 := crc16_local(input)
	result2 := crc16_local(input)

	if result1 != result2 {
		t.Errorf("CRC16 consistency failed: first=%X, second=%X", result1, result2)
	}
}

func TestCRC16_EmptyInput(t *testing.T) {
	// 验证空输入返回默认值 0xFFFF
	result := crc16_local([]byte{})
	if result != 0xFFFF {
		t.Errorf("CRC16 empty input = 0x%04X, want 0xFFFF", result)
	}
}

func TestCRC16_Deterministic(t *testing.T) {
	// 验证 CRC16 是确定性的
	tests := []struct {
		name  string
		input []byte
	}{
		{"single byte", []byte{0x01}},
		{"two bytes", []byte{0x01, 0x02}},
		{"frame header", []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x02}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 运行多次确保一致性
			for i := 0; i < 10; i++ {
				result := crc16_local(tt.input)
				if i > 0 {
					prevResult := crc16_local(tt.input)
					if result != prevResult {
						t.Errorf("CRC16 not deterministic: first=%X, second=%X", result, prevResult)
					}
				}
			}
		})
	}
}

// =============================================================================
// Test: CRC 校验函数
// =============================================================================

func TestCheckCRC_Valid(t *testing.T) {
	// 构建一个带有正确 CRC 的帧
	addr := byte(0x01)
	start := uint16(0x0000)
	qty := uint16(0x02)

	frame := buildReadFrame_local(addr, start, qty)

	if !checkCRC_local(frame) {
		t.Error("Valid CRC check failed for built frame")
	}
}

func TestCheckCRC_GeneratedFrame(t *testing.T) {
	// 测试不同参数构建的帧
	tests := []struct {
		name  string
		addr  byte
		start uint16
		qty   uint16
	}{
		{"default", 0x01, 0x0000, 0x02},
		{"addr 10", 0x0A, 0x0001, 0x01},
		{"addr 255", 0xFF, 0x0100, 0x10},
		{"many regs", 0x01, 0x0000, 0x7D}, // 125 个寄存器
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := buildReadFrame_local(tt.addr, tt.start, tt.qty)
			if !checkCRC_local(frame) {
				t.Error("CRC validation failed for built frame")
			}
		})
	}
}

func TestCheckCRC_Invalid(t *testing.T) {
	// 错误的 CRC
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x01, 0x00, 0x00}

	if checkCRC_local(data) {
		t.Error("Invalid CRC should return false")
	}
}

func TestCheckCRC_TooShort(t *testing.T) {
	data := []byte{0x01, 0x03}

	if checkCRC_local(data) {
		t.Error("Too short data should return false")
	}
}

func TestCheckCRC_Empty(t *testing.T) {
	if checkCRC_local([]byte{}) {
		t.Error("Empty data should return false")
	}
}

func TestCheckCRC_SingleByte(t *testing.T) {
	if checkCRC_local([]byte{0x00, 0x00}) {
		t.Error("Single byte data should return false")
	}
}

func TestCheckCRC_ModifiedData(t *testing.T) {
	// 修改数据后 CRC 应该失败
	frame := buildReadFrame_local(0x01, 0x0000, 0x02)

	// 修改一个字节
	original := frame[3]
	frame[3] = 0xFF

	if checkCRC_local(frame) {
		t.Error("Modified frame should have invalid CRC")
	}

	// 恢复后应该正确
	frame[3] = original
	if !checkCRC_local(frame) {
		t.Error("Restored frame should have valid CRC")
	}
}

// =============================================================================
// Test: 构建读请求帧
// =============================================================================

func TestBuildReadFrame_Basic(t *testing.T) {
	addr := byte(0x01)
	start := uint16(0x0000)
	qty := uint16(0x02)

	frame := buildReadFrame_local(addr, start, qty)

	// 验证帧长度
	if len(frame) != 8 {
		t.Errorf("Frame length = %d, want 8", len(frame))
	}

	// 验证从站地址
	if frame[0] != addr {
		t.Errorf("Address = 0x%02X, want 0x%02X", frame[0], addr)
	}

	// 验证功能码
	if frame[1] != 0x03 {
		t.Errorf("Function code = 0x%02X, want 0x03", frame[1])
	}

	// 验证起始地址
	if frame[2] != 0x00 || frame[3] != 0x00 {
		t.Errorf("Start address = 0x%02X%02X, want 0x0000", frame[2], frame[3])
	}

	// 验证寄存器数量
	if frame[4] != 0x00 || frame[5] != 0x02 {
		t.Errorf("Quantity = 0x%02X%02X, want 0x0002", frame[4], frame[5])
	}

	// 验证 CRC
	if !checkCRC_local(frame) {
		t.Error("Frame has invalid CRC")
	}
}

func TestBuildReadFrame_DifferentAddress(t *testing.T) {
	tests := []struct {
		addr byte
	}{
		{0x01},
		{0x0A},
		{0xFF},
	}

	for _, tt := range tests {
		frame := buildReadFrame_local(tt.addr, 0x0000, 1)
		if frame[0] != tt.addr {
			t.Errorf("Address = 0x%02X, want 0x%02X", frame[0], tt.addr)
		}
	}
}

func TestBuildReadFrame_DifferentStartAddress(t *testing.T) {
	tests := []struct {
		name  string
		start uint16
	}{
		{"0x0000", 0x0000},
		{"0x0001", 0x0001},
		{"0x0100", 0x0100},
		{"0xFFFF", 0xFFFF},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := buildReadFrame_local(0x01, tt.start, 1)
			if frame[2] != byte(tt.start>>8) || frame[3] != byte(tt.start) {
				t.Errorf("Start address bytes = 0x%02X%02X, want 0x%04X", frame[2], frame[3], tt.start)
			}
		})
	}
}

func TestBuildReadFrame_DifferentQuantity(t *testing.T) {
	tests := []struct {
		qty uint16
	}{
		{1},
		{2},
		{10},
		{125},
	}

	for _, tt := range tests {
		frame := buildReadFrame_local(0x01, 0x0000, tt.qty)
		if frame[4] != byte(tt.qty>>8) || frame[5] != byte(tt.qty) {
			t.Errorf("Quantity bytes = 0x%02X%02X, want 0x%04X", frame[4], frame[5], tt.qty)
		}
	}
}

func TestBuildReadFrame_CRCIsCorrect(t *testing.T) {
	tests := []struct {
		addr  byte
		start uint16
		qty   uint16
	}{
		{0x01, 0x0000, 0x02},
		{0x0A, 0x0001, 0x01},
		{0xFF, 0x0100, 0x10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			frame := buildReadFrame_local(tt.addr, tt.start, tt.qty)
			if !checkCRC_local(frame) {
				t.Error("CRC validation failed")
			}
		})
	}
}

func TestBuildReadFrame_FrameLength(t *testing.T) {
	// 验证帧长度始终为 8 字节
	tests := []struct {
		qty uint16
	}{
		{1},
		{10},
		{100},
	}

	for _, tt := range tests {
		frame := buildReadFrame_local(0x01, 0x0000, tt.qty)
		if len(frame) != 8 {
			t.Errorf("Frame length = %d, want 8 for qty=%d", len(frame), tt.qty)
		}
	}
}

// =============================================================================
// Test: 解析读响应
// =============================================================================

func TestParseReadResponse_Valid(t *testing.T) {
	// 构建一个有效的响应帧
	addr := byte(0x01)
	values := []uint16{0x0064, 0x0001}
	resp := buildMockResponse_local(addr, values)

	parsedValues, err := parseReadResponse_local(resp, addr)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(parsedValues) != len(values) {
		t.Errorf("Values length = %d, want %d", len(parsedValues), len(values))
	}
	for i := range values {
		if parsedValues[i] != values[i] {
			t.Errorf("values[%d] = 0x%04X, want 0x%04X", i, parsedValues[i], values[i])
		}
	}
}

func TestParseReadResponse_SingleRegister(t *testing.T) {
	addr := byte(0x01)
	values := []uint16{0x012B}
	resp := buildMockResponse_local(addr, values)

	parsedValues, err := parseReadResponse_local(resp, addr)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(parsedValues) != 1 {
		t.Errorf("Values length = %d, want 1", len(parsedValues))
	}
	if parsedValues[0] != 0x012B {
		t.Errorf("values[0] = 0x%04X, want 0x012B", parsedValues[0])
	}
}

func TestParseReadResponse_ZeroValues(t *testing.T) {
	addr := byte(0x01)
	values := []uint16{0x0000}
	resp := buildMockResponse_local(addr, values)

	parsedValues, err := parseReadResponse_local(resp, addr)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if parsedValues[0] != 0x0000 {
		t.Errorf("values[0] = 0x%04X, want 0x0000", parsedValues[0])
	}
}

func TestParseReadResponse_MaxValues(t *testing.T) {
	addr := byte(0x01)
	values := []uint16{0xFFFF}
	resp := buildMockResponse_local(addr, values)

	parsedValues, err := parseReadResponse_local(resp, addr)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if parsedValues[0] != 0xFFFF {
		t.Errorf("values[0] = 0x%04X, want 0xFFFF", parsedValues[0])
	}
}

func TestParseReadResponse_WrongAddress(t *testing.T) {
	addr := byte(0x01)
	values := []uint16{0x0064}
	resp := buildMockResponse_local(addr, values)

	// 用不同的地址解析
	_, err := parseReadResponse_local(resp, 0x02)

	if err == nil {
		t.Error("Expected error for wrong address")
	}
}

func TestParseReadResponse_WrongFunctionCode(t *testing.T) {
	// 手动构建一个错误功能码的响应
	resp := []byte{0x01, 0x04, 0x02, 0x00, 0x64, 0x00, 0x00}

	_, err := parseReadResponse_local(resp, 0x01)

	if err == nil {
		t.Error("Expected error for wrong function code")
	}
}

func TestParseReadResponse_TooShort(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"only addr", []byte{0x01}},
		{"addr and func", []byte{0x01, 0x03}},
		{"no crc", []byte{0x01, 0x03, 0x02, 0x00, 0x64}},
		{"crc only", []byte{0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseReadResponse_local(tt.data, 0x01)
			if err == nil {
				t.Error("Expected error for too short data")
			}
		})
	}
}

func TestParseReadResponse_BadCRC(t *testing.T) {
	resp := []byte{0x01, 0x03, 0x02, 0x00, 0x64, 0x00, 0x00}

	_, err := parseReadResponse_local(resp, 0x01)

	if err == nil {
		t.Error("Expected error for bad CRC")
	}
}

func TestParseReadResponse_ByteCountMismatch(t *testing.T) {
	// 字节计数不匹配
	resp := []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x00}

	_, err := parseReadResponse_local(resp, 0x01)

	if err == nil {
		t.Error("Expected error for byte count mismatch")
	}
}

func TestParseReadResponse_ByteCountZero(t *testing.T) {
	resp := []byte{0x01, 0x03, 0x00, 0x00, 0x00}

	_, err := parseReadResponse_local(resp, 0x01)

	if err == nil {
		t.Error("Expected error for zero byte count")
	}
}

func TestParseReadResponse_ManyRegisters(t *testing.T) {
	// 10 个寄存器
	addr := byte(0x01)
	values := []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	resp := buildMockResponse_local(addr, values)

	parsedValues, err := parseReadResponse_local(resp, addr)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(parsedValues) != 10 {
		t.Errorf("Values length = %d, want 10", len(parsedValues))
	}

	for i := range values {
		if parsedValues[i] != values[i] {
			t.Errorf("values[%d] = %d, want %d", i, parsedValues[i], values[i])
		}
	}
}

func TestParseReadResponse_RoundTrip(t *testing.T) {
	// 完整往返测试：构建请求 -> 构建响应 -> 解析响应
	addr := byte(0x01)
	start := uint16(0x0000)
	qty := uint16(0x02)

	// 构建请求帧
	req := buildReadFrame_local(addr, start, qty)
	if !checkCRC_local(req) {
		t.Fatal("Request frame has invalid CRC")
	}

	// 模拟传感器返回的数据
	sensorValues := []uint16{253, 555} // 温度 25.3°C, 湿度 55.5%
	resp := buildMockResponse_local(addr, sensorValues)

	// 解析响应
	parsedValues, err := parseReadResponse_local(resp, addr)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// 验证数据正确
	if len(parsedValues) != len(sensorValues) {
		t.Fatalf("Values length = %d, want %d", len(parsedValues), len(sensorValues))
	}
	for i := range sensorValues {
		if parsedValues[i] != sensorValues[i] {
			t.Errorf("values[%d] = %d, want %d", i, parsedValues[i], sensorValues[i])
		}
	}
}

// =============================================================================
// Test: 格式化浮点数
// =============================================================================

func TestFormatFloat_Basic(t *testing.T) {
	tests := []struct {
		val      float64
		decimals int
		expected string
	}{
		{0.0, 1, "0.0"},
		{1.5, 1, "1.5"},
		{123.456, 1, "123.5"},
		{123.456, 2, "123.46"},
		{123.456, 3, "123.456"},
		{-1.5, 1, "-1.5"},
		{-123.456, 2, "-123.46"},
		{0.1, 1, "0.1"},
		{0.01, 2, "0.01"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatFloat_local(tt.val, tt.decimals)
			if result != tt.expected {
				t.Errorf("formatFloat(%v, %d) = %s, want %s", tt.val, tt.decimals, result, tt.expected)
			}
		})
	}
}

func TestFormatFloat_ScaleExamples(t *testing.T) {
	// 测试实际使用场景：缩放后的值
	tests := []struct {
		name     string
		rawVal   uint16
		scale    float64
		decimals int
		expected string
	}{
		{"温度 25.3°C", 253, 0.1, 1, "25.3"},
		{"温度 25.3°C 整数", 253, 0.1, 0, "25"},
		{"湿度 55.5%", 555, 0.1, 1, "55.5"},
		{"大数值", 1234, 0.1, 1, "123.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			realVal := float64(tt.rawVal) * tt.scale
			result := formatFloat_local(realVal, tt.decimals)
			if result != tt.expected {
				t.Errorf("formatFloat(%v, %d) = %s, want %s", realVal, tt.decimals, result, tt.expected)
			}
		})
	}
}

func TestFormatFloat_ZeroDecimals(t *testing.T) {
	result := formatFloat_local(123.456, 0)
	if result != "123" {
		t.Errorf("formatFloat(123.456, 0) = %s, want 123", result)
	}
}

func TestFormatFloat_ManyDecimals(t *testing.T) {
	result := formatFloat_local(0.123456789, 10)
	if !bytes.HasPrefix([]byte(result), []byte("0.123456789")) {
		t.Errorf("formatFloat(0.123456789, 10) = %s", result)
	}
}

// =============================================================================
// Test: 错误处理
// =============================================================================

func TestErrf(t *testing.T) {
	err := errf_local("test error message")
	if err.Error() != "test error message" {
		t.Errorf("errf() = %s, want 'test error message'", err.Error())
	}
}

func TestSimpleErr_Error(t *testing.T) {
	err := simpleErr_local("simple error")
	if err.Error() != "simple error" {
		t.Errorf("simpleErr.Error() = %s, want 'simple error'", err.Error())
	}
}

func TestErrf_DifferentMessages(t *testing.T) {
	messages := []string{"error 1", "crc error", "invalid data", ""}
	for _, msg := range messages {
		err := errf_local(msg)
		if err.Error() != msg {
			t.Errorf("errf(%s) = %s", msg, err.Error())
		}
	}
}

// =============================================================================
// Test: 十六进制预览
// =============================================================================

func TestHexPreview_Normal(t *testing.T) {
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64}

	result := hexPreview_local(data, 5, 10)
	expected := "01 03 04 00 64"
	if result != expected {
		t.Errorf("hexPreview() = %s, want %s", result, expected)
	}
}

func TestHexPreview_Truncated(t *testing.T) {
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x01}

	result := hexPreview_local(data, 7, 4)
	expected := "01 03 04 00"
	if result != expected {
		t.Errorf("hexPreview() = %s, want %s", result, expected)
	}
}

func TestHexPreview_ZeroN(t *testing.T) {
	data := []byte{0x01, 0x03}

	result := hexPreview_local(data, 0, 10)
	if result != "" {
		t.Errorf("hexPreview(n=0) = %s, want empty string", result)
	}
}

func TestHexPreview_NegativeN(t *testing.T) {
	data := []byte{0x01, 0x03}

	result := hexPreview_local(data, -1, 10)
	if result != "" {
		t.Errorf("hexPreview(n=-1) = %s, want empty string", result)
	}
}

func TestHexPreview_ExceedsLength(t *testing.T) {
	data := []byte{0x01, 0x03, 0x04}

	result := hexPreview_local(data, 10, 10)
	expected := "01 03 04"
	if result != expected {
		t.Errorf("hexPreview() = %s, want %s", result, expected)
	}
}

func TestHexPreview_Format(t *testing.T) {
	// 验证十六进制格式正确
	data := []byte{0x0A, 0x0B, 0x0C}
	result := hexPreview_local(data, 3, 10)
	// 应该包含空格分隔
	if len(result) < 5 {
		t.Errorf("hexPreview too short: %s", result)
	}
}

// =============================================================================
// Test: 转义字符串
// =============================================================================

func TestAppendEscaped_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"hello world", "hello world"},
		{"中文测试", "中文测试"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := string(appendEscaped_local([]byte{}, tt.input))
			if result != tt.expected {
				t.Errorf("appendEscaped(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAppendEscaped_SpecialChars(t *testing.T) {
	result := string(appendEscaped_local([]byte{}, `hello"world`))
	expected := `hello\"world`
	if result != expected {
		t.Errorf("appendEscaped() = %s, want %s", result, expected)
	}

	result = string(appendEscaped_local([]byte{}, `path\to\file`))
	expected = `path\\to\\file`
	if result != expected {
		t.Errorf("appendEscaped() = %s, want %s", result, expected)
	}
}

func TestAppendEscaped_Mixed(t *testing.T) {
	input := `hello "world" and \path\`
	result := string(appendEscaped_local([]byte{}, input))
	expected := `hello \"world\" and \\path\\`
	if result != expected {
		t.Errorf("appendEscaped() = %s, want %s", result, expected)
	}
}

func TestAppendEscaped_OnlyQuotes(t *testing.T) {
	result := string(appendEscaped_local([]byte{}, `"`))
	expected := `\"`
	if result != expected {
		t.Errorf("appendEscaped(\\\"\\\") = %s, want %s", result, expected)
	}
}

func TestAppendEscaped_OnlyBackslash(t *testing.T) {
	result := string(appendEscaped_local([]byte{}, `\`))
	expected := `\\`
	if result != expected {
		t.Errorf("appendEscaped(\\\\) = %s, want %s", result, expected)
	}
}

// =============================================================================
// Test: 综合集成测试
// =============================================================================

func TestIntegration_ReadFrameAndParse(t *testing.T) {
	// 集成测试：构建请求帧 -> 模拟响应 -> 解析响应
	addr := byte(0x01)
	startAddr := uint16(0x0000)
	qty := uint16(0x02)

	// 1. 构建请求帧
	reqFrame := buildReadFrame_local(addr, startAddr, qty)
	if len(reqFrame) != 8 {
		t.Fatalf("Request frame length = %d, want 8", len(reqFrame))
	}

	// 2. 模拟响应数据 (假设温度 25.3°C = 253, 湿度 55.5% = 555)
	sensorValues := []uint16{253, 555}
	respData := buildMockResponse_local(addr, sensorValues)

	// 3. 解析响应
	values, err := parseReadResponse_local(respData, addr)
	if err != nil {
		t.Fatalf("parseReadResponse error: %v", err)
	}

	// 4. 验证解析结果
	if len(values) != 2 {
		t.Fatalf("values length = %d, want 2", len(values))
	}

	expectedTemp := uint16(253) // 25.3°C * 10
	expectedHum := uint16(555)  // 55.5% * 10

	if values[0] != expectedTemp {
		t.Errorf("temperature value = %d, want %d", values[0], expectedTemp)
	}
	if values[1] != expectedHum {
		t.Errorf("humidity value = %d, want %d", values[1], expectedHum)
	}

	// 5. 计算实际值
	tempReal := float64(values[0]) * 0.1
	humReal := float64(values[1]) * 0.1

	tempStr := formatFloat_local(tempReal, 1)
	humStr := formatFloat_local(humReal, 1)

	if tempStr != "25.3" {
		t.Errorf("temperature string = %s, want '25.3'", tempStr)
	}
	if humStr != "55.5" {
		t.Errorf("humidity string = %s, want '55.5'", humStr)
	}
}

func TestIntegration_MultipleDevices(t *testing.T) {
	// 测试多个设备地址
	devices := []byte{0x01, 0x0A, 0x0B, 0x0C}

	for _, devAddr := range devices {
		frame := buildReadFrame_local(devAddr, 0x0000, 1)
		if frame[0] != devAddr {
			t.Errorf("Device 0x%02X: address byte = 0x%02X", devAddr, frame[0])
		}

		// 验证 CRC
		if !checkCRC_local(frame) {
			t.Errorf("Device 0x%02X: CRC validation failed", devAddr)
		}
	}
}

func TestIntegration_FullFrameValidation(t *testing.T) {
	// 完整帧验证测试
	tests := []struct {
		name   string
		addr   byte
		values []uint16
	}{
		{
			name:   "温度+湿度",
			addr:   0x01,
			values: []uint16{253, 555},
		},
		{
			name:   "仅温度",
			addr:   0x02,
			values: []uint16{250},
		},
		{
			name:   "仅湿度",
			addr:   0x03,
			values: []uint16{600},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 构建请求
			req := buildReadFrame_local(tt.addr, 0x0000, uint16(len(tt.values)))

			// 验证请求帧
			if !checkCRC_local(req) {
				t.Fatal("Request CRC validation failed")
			}

			// 构建响应
			resp := buildMockResponse_local(tt.addr, tt.values)

			// 解析响应
			parsedValues, err := parseReadResponse_local(resp, tt.addr)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// 验证解析结果
			if len(parsedValues) != len(tt.values) {
				t.Fatalf("Values length = %d, want %d", len(parsedValues), len(tt.values))
			}

			for i, v := range tt.values {
				if parsedValues[i] != v {
					t.Errorf("values[%d] = %d, want %d", i, parsedValues[i], v)
				}
			}
		})
	}
}

func TestIntegration_TemperatureAndHumidity(t *testing.T) {
	// 测试温湿度传感器的典型场景
	// 假设：设备地址 0x01，温度 23.5°C，湿度 58.2%

	addr := byte(0x01)
	rawTemp := uint16(235) // 23.5 * 10
	rawHum := uint16(582)  // 58.2 * 10

	// 1. 构建读取请求
	req := buildReadFrame_local(addr, 0x0000, 2)
	if !checkCRC_local(req) {
		t.Fatal("Request frame has invalid CRC")
	}

	// 2. 模拟传感器响应
	sensorValues := []uint16{rawTemp, rawHum}
	resp := buildMockResponse_local(addr, sensorValues)

	// 3. 解析响应
	values, err := parseReadResponse_local(resp, addr)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// 4. 验证原始值
	if values[0] != rawTemp {
		t.Errorf("Raw temperature = %d, want %d", values[0], rawTemp)
	}
	if values[1] != rawHum {
		t.Errorf("Raw humidity = %d, want %d", values[1], rawHum)
	}

	// 5. 转换为实际值并格式化
	tempReal := float64(values[0]) * 0.1
	humReal := float64(values[1]) * 0.1

	tempStr := formatFloat_local(tempReal, 1)
	humStr := formatFloat_local(humReal, 1)

	if tempStr != "23.5" {
		t.Errorf("Temperature = %s, want '23.5'", tempStr)
	}
	if humStr != "58.2" {
		t.Errorf("Humidity = %s, want '58.2'", humStr)
	}
}

// =============================================================================
// 辅助函数：构建模拟响应
// =============================================================================

func buildMockResponse_local(addr byte, values []uint16) []byte {
	byteCount := len(values) * 2
	resp := make([]byte, 3+byteCount+2)

	resp[0] = addr
	resp[1] = 0x03 // 功能码
	resp[2] = byte(byteCount)

	for i, v := range values {
		resp[3+i*2] = byte(v >> 8)
		resp[4+i*2] = byte(v)
	}

	crc := crc16_local(resp[:3+byteCount])
	resp[3+byteCount] = byte(crc)
	resp[4+byteCount] = byte(crc >> 8)

	return resp
}
