// =============================================================================
// 温湿度传感器 - Modbus RTU 驱动 单元测试
// =============================================================================
//
// 测试策略：
// - 只测试纯逻辑函数 (CRC, 帧构建, 解析, 格式化)
// - 排除依赖 WebAssembly/PDK 的函数
//
// =============================================================================
package main

import (
	"bytes"
	"testing"
)

// =============================================================================
// Test: CRC16 校验函数
// =============================================================================

func TestCRC16(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint16
	}{
		{
			name:     "空数据",
			input:    []byte{},
			expected: 0xFFFF,
		},
		{
			name:     "单个字节 0x00",
			input:    []byte{0x00},
			expected: 0xFFFF,
		},
		{
			name:     "单个字节 0xFF",
			input:    []byte{0xFF},
			expected: 0xFF1F,
		},
		{
			name:     "Modbus 地址 0x01",
			input:    []byte{0x01},
			expected: 0xFF21,
		},
		{
			name:     "功能码 0x03",
			input:    []byte{0x03},
			expected: 0xFF23,
		},
		{
			name:     "地址 + 功能码",
			input:    []byte{0x01, 0x03},
			expected: 0xFF16,
		},
		{
			name:     "完整帧头 6 字节",
			input:    []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x02},
			expected: 0xC407,
		},
		{
			name:     "寄存器地址 0x0000",
			input:    []byte{0x00, 0x00},
			expected: 0xFFFF,
		},
		{
			name:     "寄存器地址 0x0001",
			input:    []byte{0x00, 0x01},
			expected: 0xFFFE,
		},
		{
			name:     "多字节数据",
			input:    []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x01},
			expected: 0xC241,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := crc16(tt.input)
			if result != tt.expected {
				t.Errorf("crc16(%v) = 0x%04X, want 0x%04X", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCRC16_Consistency(t *testing.T) {
	// 验证 CRC16 的一致性（同输入同输出）
	input := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x02}

	result1 := crc16(input)
	result2 := crc16(input)

	if result1 != result2 {
		t.Errorf("CRC16 consistency failed: first=%X, second=%X", result1, result2)
	}
}

func TestCRC16_EmptyInput(t *testing.T) {
	// 验证空输入返回默认值
	result := crc16([]byte{})
	if result != 0xFFFF {
		t.Errorf("CRC16 empty input = 0x%04X, want 0xFFFF", result)
	}
}

// =============================================================================
// Test: CRC 校验函数
// =============================================================================

func TestCheckCRC_Valid(t *testing.T) {
	// 构造一个带有正确 CRC 的帧
	// 帧: 01 03 04 00 64 00 01 CRC
	// CRC of [01 03 04 00 64 00 01] = 0xC241
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x01, 0x41, 0xC2}

	if !checkCRC(data) {
		t.Error("Valid CRC check failed")
	}
}

func TestCheckCRC_Invalid(t *testing.T) {
	// 错误的 CRC
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x01, 0x00, 0x00}

	if checkCRC(data) {
		t.Error("Invalid CRC should return false")
	}
}

func TestCheckCRC_TooShort(t *testing.T) {
	// 数据太短，无法包含 CRC
	data := []byte{0x01, 0x03}

	if checkCRC(data) {
		t.Error("Too short data should return false")
	}
}

func TestCheckCRC_Empty(t *testing.T) {
	// 空数据
	if checkCRC([]byte{}) {
		t.Error("Empty data should return false")
	}
}

func TestCheckCRC_SingleByte(t *testing.T) {
	// 只有 CRC 部分，没有实际数据
	if checkCRC([]byte{0x00, 0x00}) {
		t.Error("Single byte data should return false")
	}
}

// =============================================================================
// Test: 构建读请求帧
// =============================================================================

func TestBuildReadFrame_Basic(t *testing.T) {
	addr := byte(0x01)
	start := uint16(0x0000)
	qty := uint16(0x02)

	frame := buildReadFrame(addr, start, qty)

	// 验证帧长度
	if len(frame) != 8 {
		t.Errorf("Frame length = %d, want 8", len(frame))
	}

	// 验证从站地址
	if frame[0] != addr {
		t.Errorf("Address = 0x%02X, want 0x%02X", frame[0], addr)
	}

	// 验证功能码
	if frame[1] != FUNC_CODE_READ {
		t.Errorf("Function code = 0x%02X, want 0x%02X", frame[1], FUNC_CODE_READ)
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
	expectedCRC := crc16(frame[:6])
	gotCRC := uint16(frame[6]) | uint16(frame[7])<<8
	if gotCRC != expectedCRC {
		t.Errorf("CRC = 0x%04X, want 0x%04X", gotCRC, expectedCRC)
	}
}

func TestBuildReadFrame_DifferentAddress(t *testing.T) {
	// 测试不同的从站地址
	tests := []struct {
		addr byte
	}{
		{0x01},
		{0x0A},
		{0xFF},
	}

	for _, tt := range tests {
		frame := buildReadFrame(tt.addr, 0x0000, 1)
		if frame[0] != tt.addr {
			t.Errorf("Address = 0x%02X, want 0x%02X", frame[0], tt.addr)
		}
	}
}

func TestBuildReadFrame_DifferentStartAddress(t *testing.T) {
	// 测试不同的起始地址
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
			frame := buildReadFrame(0x01, tt.start, 1)
			if frame[2] != byte(tt.start>>8) || frame[3] != byte(tt.start) {
				t.Errorf("Start address bytes = 0x%02X%02X, want 0x%04X", frame[2], frame[3], tt.start)
			}
		})
	}
}

func TestBuildReadFrame_DifferentQuantity(t *testing.T) {
	// 测试不同的寄存器数量
	tests := []struct {
		qty uint16
	}{
		{1},
		{2},
		{10},
		{125},
	}

	for _, tt := range tests {
		frame := buildReadFrame(0x01, 0x0000, tt.qty)
		if frame[4] != byte(tt.qty>>8) || frame[5] != byte(tt.qty) {
			t.Errorf("Quantity bytes = 0x%02X%02X, want 0x%04X", frame[4], frame[5], tt.qty)
		}
	}
}

func TestBuildReadFrame_CRCIsCorrect(t *testing.T) {
	// 验证不同参数的 CRC 计算正确性
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
			frame := buildReadFrame(tt.addr, tt.start, tt.qty)
			if !checkCRC(frame) {
				t.Error("CRC validation failed")
			}
		})
	}
}

// =============================================================================
// Test: 解析读响应
// =============================================================================

func TestParseReadResponse_Valid(t *testing.T) {
	// 构造有效响应: 01 03 04 00 64 00 01 CRC
	// 功能码 0x03, 4 字节数据: 0x0064, 0x0001
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x01, 0x41, 0xC2}

	values, err := parseReadResponse(data, 0x01)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(values) != 2 {
		t.Errorf("Values length = %d, want 2", len(values))
	}
	if values[0] != 0x0064 {
		t.Errorf("values[0] = 0x%04X, want 0x0064", values[0])
	}
	if values[1] != 0x0001 {
		t.Errorf("values[1] = 0x%04X, want 0x0001", values[1])
	}
}

func TestParseReadResponse_SingleRegister(t *testing.T) {
	// 单个寄存器响应: 01 03 02 01 2B CRC
	data := []byte{0x01, 0x03, 0x02, 0x01, 0x2B, 0x39, 0x90}

	values, err := parseReadResponse(data, 0x01)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(values) != 1 {
		t.Errorf("Values length = %d, want 1", len(values))
	}
	if values[0] != 0x012B {
		t.Errorf("values[0] = 0x%04X, want 0x012B", values[0])
	}
}

func TestParseReadResponse_ZeroValues(t *testing.T) {
	// 零值响应
	data := []byte{0x01, 0x03, 0x02, 0x00, 0x00, 0xC4, 0x0A}

	values, err := parseReadResponse(data, 0x01)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if values[0] != 0x0000 {
		t.Errorf("values[0] = 0x%04X, want 0x0000", values[0])
	}
}

func TestParseReadResponse_MaxValues(t *testing.T) {
	// 最大值响应: 0xFFFF
	data := []byte{0x01, 0x03, 0x02, 0xFF, 0xFF, 0xF8, 0x16}

	values, err := parseReadResponse(data, 0x01)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if values[0] != 0xFFFF {
		t.Errorf("values[0] = 0x%04X, want 0xFFFF", values[0])
	}
}

func TestParseReadResponse_WrongAddress(t *testing.T) {
	// 错误的从站地址
	data := []byte{0x02, 0x03, 0x02, 0x00, 0x64, 0x00, 0x00}

	_, err := parseReadResponse(data, 0x01)

	if err == nil {
		t.Error("Expected error for wrong address")
	}
}

func TestParseReadResponse_WrongFunctionCode(t *testing.T) {
	// 错误的功能码
	data := []byte{0x01, 0x04, 0x02, 0x00, 0x64, 0x00, 0x00}

	_, err := parseReadResponse(data, 0x01)

	if err == nil {
		t.Error("Expected error for wrong function code")
	}
}

func TestParseReadResponse_TooShort(t *testing.T) {
	// 数据太短
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
			_, err := parseReadResponse(tt.data, 0x01)
			if err == nil {
				t.Error("Expected error for too short data")
			}
		})
	}
}

func TestParseReadResponse_BadCRC(t *testing.T) {
	// CRC 错误
	data := []byte{0x01, 0x03, 0x02, 0x00, 0x64, 0x00, 0x00}

	_, err := parseReadResponse(data, 0x01)

	if err == nil {
		t.Error("Expected error for bad CRC")
	}
}

func TestParseReadResponse_ByteCountMismatch(t *testing.T) {
	// 字节计数不匹配 (声明 4 字节但实际只有 3 字节)
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x00}

	_, err := parseReadResponse(data, 0x01)

	if err == nil {
		t.Error("Expected error for byte count mismatch")
	}
}

func TestParseReadResponse_ByteCountZero(t *testing.T) {
	// 字节计数为 0
	data := []byte{0x01, 0x03, 0x00, 0x00, 0x00}

	_, err := parseReadResponse(data, 0x01)

	if err == nil {
		t.Error("Expected error for zero byte count")
	}
}

func TestParseReadResponse_ManyRegisters(t *testing.T) {
	// 多个寄存器: 10 个寄存器 = 20 字节数据
	data := []byte{
		0x01, 0x03, 0x14, // 地址, 功能码, 字节计数
		0x00, 0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x04, // 寄存器 1-4
		0x00, 0x05, 0x00, 0x06, 0x00, 0x07, 0x00, 0x08, // 寄存器 5-8
		0x00, 0x09, 0x00, 0x0A, // 寄存器 9-10
		0x00, 0x00, // CRC (placeholder)
	}

	// 计算正确 CRC
	crc := crc16(data[:3+20])
	data[23] = byte(crc)
	data[24] = byte(crc >> 8)

	values, err := parseReadResponse(data, 0x01)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(values) != 10 {
		t.Errorf("Values length = %d, want 10", len(values))
	}

	// 验证每个值
	for i := 0; i < 10; i++ {
		expected := uint16(i + 1)
		if values[i] != expected {
			t.Errorf("values[%d] = 0x%04X, want 0x%04X", i, values[i], expected)
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
			result := formatFloat(tt.val, tt.decimals)
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
			result := formatFloat(realVal, tt.decimals)
			if result != tt.expected {
				t.Errorf("formatFloat(%v, %d) = %s, want %s", realVal, tt.decimals, result, tt.expected)
			}
		})
	}
}

func TestFormatFloat_ZeroDecimals(t *testing.T) {
	result := formatFloat(123.456, 0)
	if result != "123" {
		t.Errorf("formatFloat(123.456, 0) = %s, want 123", result)
	}
}

func TestFormatFloat_ManyDecimals(t *testing.T) {
	// 大量小数位
	result := formatFloat(0.123456789, 10)
	if !bytes.HasPrefix([]byte(result), []byte("0.123456789")) {
		t.Errorf("formatFloat(0.123456789, 10) = %s", result)
	}
}

// =============================================================================
// Test: 错误处理
// =============================================================================

func TestErrf(t *testing.T) {
	err := errf("test error message")
	if err.Error() != "test error message" {
		t.Errorf("errf() = %s, want 'test error message'", err.Error())
	}
}

func TestSimpleErr_Error(t *testing.T) {
	err := simpleErr("simple error")
	if err.Error() != "simple error" {
		t.Errorf("simpleErr.Error() = %s, want 'simple error'", err.Error())
	}
}

// =============================================================================
// Test: 十六进制预览
// =============================================================================

func TestHexPreview_Normal(t *testing.T) {
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64}

	result := hexPreview(data, 5, 10)
	expected := "01 03 04 00 64"
	if result != expected {
		t.Errorf("hexPreview() = %s, want %s", result, expected)
	}
}

func TestHexPreview_Truncated(t *testing.T) {
	data := []byte{0x01, 0x03, 0x04, 0x00, 0x64, 0x00, 0x01}

	result := hexPreview(data, 7, 4)
	expected := "01 03 04 00"
	if result != expected {
		t.Errorf("hexPreview() = %s, want %s", result, expected)
	}
}

func TestHexPreview_ZeroN(t *testing.T) {
	data := []byte{0x01, 0x03}

	result := hexPreview(data, 0, 10)
	if result != "" {
		t.Errorf("hexPreview(n=0) = %s, want empty string", result)
	}
}

func TestHexPreview_NegativeN(t *testing.T) {
	data := []byte{0x01, 0x03}

	result := hexPreview(data, -1, 10)
	if result != "" {
		t.Errorf("hexPreview(n=-1) = %s, want empty string", result)
	}
}

func TestHexPreview_ExceedsLength(t *testing.T) {
	data := []byte{0x01, 0x03, 0x04}

	result := hexPreview(data, 10, 10)
	expected := "01 03 04"
	if result != expected {
		t.Errorf("hexPreview() = %s, want %s", result, expected)
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
			result := string(appendEscaped([]byte{}, tt.input))
			if result != tt.expected {
				t.Errorf("appendEscaped(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAppendEscaped_SpecialChars(t *testing.T) {
	// 测试转义字符
	result := string(appendEscaped([]byte{}, `hello"world`))
	expected := `hello\"world`
	if result != expected {
		t.Errorf("appendEscaped() = %s, want %s", result, expected)
	}

	result = string(appendEscaped([]byte{}, `path\to\file`))
	expected = `path\\to\\file`
	if result != expected {
		t.Errorf("appendEscaped() = %s, want %s", result, expected)
	}
}

func TestAppendEscaped_Mixed(t *testing.T) {
	input := `hello "world" and \path\`
	result := string(appendEscaped([]byte{}, input))
	expected := `hello \"world\" and \\path\\`
	if result != expected {
		t.Errorf("appendEscaped() = %s, want %s", result, expected)
	}
}

// =============================================================================
// Test: 常量定义
// =============================================================================

func TestDriverVersion(t *testing.T) {
	if DriverVersion != "1.0.0" {
		t.Errorf("DriverVersion = %s, want '1.0.0'", DriverVersion)
	}
}

func TestFunctionCodes(t *testing.T) {
	if FUNC_CODE_READ != 0x03 {
		t.Errorf("FUNC_CODE_READ = 0x%02X, want 0x03", FUNC_CODE_READ)
	}
	if FUNC_CODE_WRITE != 0x06 {
		t.Errorf("FUNC_CODE_WRITE = 0x%02X, want 0x06", FUNC_CODE_WRITE)
	}
}

func TestRegisterAddresses(t *testing.T) {
	if REG_TEMPERATURE != 0x0000 {
		t.Errorf("REG_TEMPERATURE = 0x%04X, want 0x0000", REG_TEMPERATURE)
	}
	if REG_HUMIDITY != 0x0001 {
		t.Errorf("REG_HUMIDITY = 0x%04X, want 0x0001", REG_HUMIDITY)
	}
}

// =============================================================================
// Test: 点表配置
// =============================================================================

func TestPointConfig_Fields(t *testing.T) {
	if len(pointConfig) != 2 {
		t.Errorf("pointConfig length = %d, want 2", len(pointConfig))
	}

	// 验证温度点配置
	tempCfg := pointConfig[0]
	if tempCfg.Field != "temperature" {
		t.Errorf("temperature field = %s, want 'temperature'", tempCfg.Field)
	}
	if tempCfg.Address != REG_TEMPERATURE {
		t.Errorf("temperature address = 0x%04X, want 0x%04X", tempCfg.Address, REG_TEMPERATURE)
	}
	if tempCfg.Scale != 0.1 {
		t.Errorf("temperature scale = %f, want 0.1", tempCfg.Scale)
	}
	if tempCfg.Decimals != 1 {
		t.Errorf("temperature decimals = %d, want 1", tempCfg.Decimals)
	}
	if tempCfg.Unit != "°C" {
		t.Errorf("temperature unit = %s, want '°C'", tempCfg.Unit)
	}

	// 验证湿度点配置
	humCfg := pointConfig[1]
	if humCfg.Field != "humidity" {
		t.Errorf("humidity field = %s, want 'humidity'", humCfg.Field)
	}
	if humCfg.Address != REG_HUMIDITY {
		t.Errorf("humidity address = 0x%04X, want 0x%04X", humCfg.Address, REG_HUMIDITY)
	}
	if humCfg.Scale != 0.1 {
		t.Errorf("humidity scale = %f, want 0.1", humCfg.Scale)
	}
	if humCfg.Unit != "%" {
		t.Errorf("humidity unit = %s, want '%%'", humCfg.Unit)
	}
}

// =============================================================================
// Test: DriverConfig
// =============================================================================

func TestDriverConfig_Default(t *testing.T) {
	cfg := DriverConfig{}

	// 验证零值
	if cfg.DeviceAddress != 0 {
		t.Errorf("default DeviceAddress = %d, want 0", cfg.DeviceAddress)
	}
	if cfg.FuncName != "" {
		t.Errorf("default FuncName = %s, want empty", cfg.FuncName)
	}
	if cfg.FieldName != "" {
		t.Errorf("default FieldName = %s, want empty", cfg.FieldName)
	}
	if cfg.Value != "" {
		t.Errorf("default Value = %s, want empty", cfg.Value)
	}
	if cfg.Debug != false {
		t.Errorf("default Debug = %v, want false", cfg.Debug)
	}
}

func TestDriverConfig_JSONTags(t *testing.T) {
	cfg := DriverConfig{
		DeviceAddress: 1,
		FuncName:      "read",
		FieldName:     "temperature",
		Value:         "25.3",
		Debug:         true,
	}

	// 验证 JSON 标签对应的字段值
	if cfg.DeviceAddress != 1 {
		t.Errorf("DeviceAddress = %d, want 1", cfg.DeviceAddress)
	}
	if cfg.FuncName != "read" {
		t.Errorf("FuncName = %s, want 'read'", cfg.FuncName)
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
	reqFrame := buildReadFrame(addr, startAddr, qty)
	if len(reqFrame) != 8 {
		t.Fatalf("Request frame length = %d, want 8", len(reqFrame))
	}

	// 2. 模拟响应数据 (假设温度 25.3°C = 253, 湿度 55.5% = 555)
	// 响应: 01 03 04 00 FD 02 2B CRC
	respData := []byte{0x01, 0x03, 0x04, 0x00, 0xFD, 0x02, 0x2B, 0x00, 0x00}
	crc := crc16(respData[:7])
	respData[7] = byte(crc)
	respData[8] = byte(crc >> 8)

	// 3. 解析响应
	values, err := parseReadResponse(respData, addr)
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

	tempStr := formatFloat(tempReal, 1)
	humStr := formatFloat(humReal, 1)

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
		frame := buildReadFrame(devAddr, REG_TEMPERATURE, 1)
		if frame[0] != devAddr {
			t.Errorf("Device 0x%02X: address byte = 0x%02X", devAddr, frame[0])
		}

		// 验证 CRC
		if !checkCRC(frame) {
			t.Errorf("Device 0x%02X: CRC validation failed", devAddr)
		}
	}
}

func TestIntegration_FullFrameValidation(t *testing.T) {
	// 完整帧验证测试
	tests := []struct {
		name    string
		addr    byte
		start   uint16
		qty     uint16
		values  []uint16
	}{
		{
			name:   "温度+湿度",
			addr:   0x01,
			start:  0x0000,
			qty:    0x02,
			values: []uint16{253, 555},
		},
		{
			name:   "仅温度",
			addr:   0x02,
			start:  0x0000,
			qty:    0x01,
			values: []uint16{250},
		},
		{
			name:   "仅湿度",
			addr:   0x03,
			start:  0x0001,
			qty:    0x01,
			values: []uint16{600},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 构建请求
			req := buildReadFrame(tt.addr, tt.start, tt.qty)

			// 验证请求帧
			if !checkCRC(req) {
				t.Fatal("Request CRC validation failed")
			}

			// 构建响应
			resp := buildMockResponse(tt.addr, tt.values)

			// 解析响应
			parsedValues, err := parseReadResponse(resp, tt.addr)
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

// =============================================================================
// 辅助函数：构建模拟响应
// =============================================================================

func buildMockResponse(addr byte, values []uint16) []byte {
	byteCount := len(values) * 2
	resp := make([]byte, 3+byteCount+2)

	resp[0] = addr
	resp[1] = FUNC_CODE_READ
	resp[2] = byte(byteCount)

	for i, v := range values {
		resp[3+i*2] = byte(v >> 8)
		resp[4+i*2] = byte(v)
	}

	crc := crc16(resp[:3+byteCount])
	resp[3+byteCount] = byte(crc)
	resp[4+byteCount] = byte(crc >> 8)

	return resp
}
