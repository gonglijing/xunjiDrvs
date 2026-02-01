// =============================================================================
// Modbus RTU 协议包
// =============================================================================
// 纯计算实现的 Modbus RTU 协议包，不依赖具体 I/O。
// 配合 Host Functions (serial_read/serial_write) 使用。
// =============================================================================

package modbus

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// =============================================================================
// CRC-16/MODBUS 计算
// =============================================================================

// CRC16 计算 CRC-16/MODBUS 校验码
func CRC16(data []byte) uint16 {
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

// BuildRequestFrame 构建 Modbus RTU 请求帧 (用于读取)
//
// 参数:
//
//	addr: 设备地址
//	funcCode: 功能码
//	regAddr: 寄存器起始地址
//	count: 寄存器数量
//
// 返回:
//
//	完整的请求帧 (包含 2 字节 CRC)
func BuildRequestFrame(addr byte, funcCode byte, regAddr uint16, count uint16) []byte {
	// 除去 CRC 的请求帧
	req := []byte{
		addr,          // 设备地址
		funcCode,      // 功能码
		0x00,          // 寄存器地址高字节
		byte(regAddr), // 寄存器地址低字节
		0x00,          // 数量高字节
		byte(count),   // 数量低字节
	}

	// 计算 CRC
	crc := CRC16(req)
	req = append(req, byte(crc&0xFF), byte(crc>>8))

	return req
}

// BuildWriteSingleRegisterFrame 构建写单个保持寄存器的请求帧 (0x06 功能码)
//
// 参数:
//
//	addr: 设备地址
//	regAddr: 寄存器地址
//	value: 写入的值
//
// 返回:
//
//	完整的请求帧 (包含 2 字节 CRC)
func BuildWriteSingleRegisterFrame(addr byte, regAddr uint16, value uint16) []byte {
	req := []byte{
		addr,               // 设备地址
		0x06,               // 功能码: 写单个寄存器
		0x00,               // 寄存器地址高字节
		byte(regAddr),      // 寄存器地址低字节
		byte(value >> 8),   // 值高字节
		byte(value & 0xFF), // 值低字节
	}

	// 计算 CRC
	crc := CRC16(req)
	req = append(req, byte(crc&0xFF), byte(crc>>8))

	return req
}

// =============================================================================
// 响应帧解析
// =============================================================================

// ParseReadResponse 解析读取响应帧
//
// 参数:
//
//	response: 响应数据 (不含 CRC)
//	addr: 期望的设备地址
//	funcCode: 期望的功能码
//
// 返回:
//
//	[]uint16: 读取到的寄存器值数组
//	error: 解析错误
func ParseReadResponse(response []byte, addr byte, funcCode byte) ([]uint16, error) {
	// 验证地址和功能码
	if len(response) < 2 {
		return nil, ErrInvalidResponse
	}
	if response[0] != addr {
		return nil, ErrAddrMismatch
	}
	if response[1] != funcCode {
		return nil, ErrFuncCodeMismatch
	}

	// 验证数据长度
	byteCount := int(response[2])
	expectedLen := 3 + byteCount
	if len(response) < expectedLen {
		return nil, ErrInvalidResponse
	}

	// 解析寄存器值 (每2字节一个寄存器，大端序)
	values := make([]uint16, byteCount/2)
	for i := 0; i < len(values); i++ {
		offset := 3 + i*2
		values[i] = binary.BigEndian.Uint16(response[offset : offset+2])
	}

	return values, nil
}

// ParseReadResponseErr 解析异常响应
//
// 返回:
//
//	异常码 (0 表示无异常)
//	error: 如果是异常响应则返回错误
func ParseReadResponseErr(response []byte, addr byte, funcCode byte) (byte, error) {
	if len(response) < 5 {
		return 0, ErrInvalidResponse
	}

	// 异常响应功能码 = 原功能码 | 0x80
	if response[1] == funcCode|0x80 {
		return response[2], nil
	}

	return 0, nil
}

// =============================================================================
// 错误定义
// =============================================================================

var (
	ErrInvalidResponse  = ModbusError("无效的响应数据")
	ErrAddrMismatch     = ModbusError("设备地址不匹配")
	ErrFuncCodeMismatch = ModbusError("功能码不匹配")
	ErrCRCFail          = ModbusError("CRC 校验失败")
	ErrException        = ModbusError("设备返回异常")
)

// ModbusError Modbus 错误类型
type ModbusError string

func (e ModbusError) Error() string {
	return string(e)
}

// =============================================================================
// 数据类型转换工具
// =============================================================================

// Int16ToFloat64 将 int16 值按缩放因子转换
func Int16ToFloat64(value uint16, scale float64) float64 {
	return float64(int16(value)) * scale
}

// Uint16ToFloat64 将 uint16 值按缩放因子转换
func Uint16ToFloat64(value uint16, scale float64) float64 {
	return float64(value) * scale
}

// CombineInt16s 将两个 int16 合并为 int32 (大端序)
func CombineInt16s(hi, lo uint16) int32 {
	return int32(hi)<<16 | int32(lo)
}

// CombineUint16s 将两个 uint16 合并为 uint32 (大端序)
func CombineUint16s(hi, lo uint16) uint32 {
	return uint32(hi)<<16 | uint32(lo)
}

// =============================================================================
// Modbus RTU 客户端
// =============================================================================

// ReadFunc 读取函数的类型定义
type ReadFunc func(buf unsafe.Pointer, size int32) int32

// WriteFunc 写入函数的类型定义
type WriteFunc func(buf unsafe.Pointer, size int32) int32

// SleepFunc 延时函数的类型定义
type SleepFunc func(ms int32)

// ModbusRTU Modbus RTU 客户端
type ModbusRTU struct {
	addr    byte      // 设备地址
	readFn  ReadFunc  // 串口读取函数
	writeFn WriteFunc // 串口写入函数
	sleepFn SleepFunc // 延时函数
	delayMs int32     // 通信延时 (毫秒)
}

// NewModbusRTU 创建 Modbus RTU 客户端实例
//
// 参数:
//
//	addr: 设备地址
//	readFn: 串口读取函数
//	writeFn: 串口写入函数
//	sleepFn: 延时函数
//	delayMs: 读取后延时时间 (毫秒)
func NewModbusRTU(addr byte, readFn ReadFunc, writeFn WriteFunc, sleepFn SleepFunc, delayMs int32) *ModbusRTU {
	return &ModbusRTU{
		addr:    addr,
		readFn:  readFn,
		writeFn: writeFn,
		sleepFn: sleepFn,
		delayMs: delayMs,
	}
}

// ReadHoldingRegister 读取单个保持寄存器的值 (0x03 功能码)
//
// 参数:
//
//	regAddr: 寄存器地址
//
// 返回:
//
//	uint16: 读取到的寄存器值
//	error: 错误信息
func (m *ModbusRTU) ReadHoldingRegister(regAddr uint16) (uint16, error) {
	values, err := m.ReadRegisters(regAddr, 0x03, 1)
	if err != nil {
		return 0, err
	}
	return values[0], nil
}

// ReadRegisters 读取寄存器值
//
// 参数:
//
//	regAddr: 寄存器起始地址
//	funcCode: 功能码 (0x03 读取保持寄存器, 0x04 读取输入寄存器)
//	count: 寄存器数量
//
// 返回:
//
//	[]uint16: 读取到的寄存器值数组
//	error: 错误信息
func (m *ModbusRTU) ReadRegisters(regAddr uint16, funcCode byte, count uint16) ([]uint16, error) {
	// 构建请求帧
	req := BuildRequestFrame(m.addr, funcCode, regAddr, count)

	// 发送请求
	n := m.writeFn(unsafe.Pointer(&req[0]), int32(len(req)))
	if n != int32(len(req)) {
		return nil, ModbusError(fmt.Sprintf("串口写入失败: wrote %d/%d bytes", n, len(req)))
	}

	// 等待响应
	m.sleepFn(m.delayMs)

	// 读取响应
	resp := make([]byte, 64)
	n = m.readFn(unsafe.Pointer(&resp[0]), int32(len(resp)))
	if n < 5 {
		return nil, ModbusError(fmt.Sprintf("响应数据不完整，收到 %d 字节", n))
	}

	// 验证 CRC
	if !CheckCRC(resp[:n]) {
		return nil, ModbusError("CRC 校验失败")
	}

	// 解析响应 (不含 CRC)
	values, err := ParseReadResponse(resp[:n-2], m.addr, funcCode)
	if err != nil {
		return nil, err
	}

	if len(values) < int(count) {
		return nil, ModbusError("响应数据长度不足")
	}

	return values, nil
}

// WriteHoldingRegister 写入单个保持寄存器的值 (0x06 功能码)
//
// 参数:
//
//	regAddr: 寄存器地址
//	value: 写入的值
//
// 返回:
//
//	error: 错误信息
func (m *ModbusRTU) WriteHoldingRegister(regAddr uint16, value uint16) error {
	// 构建请求帧
	req := BuildWriteSingleRegisterFrame(m.addr, regAddr, value)

	// 发送请求
	n := m.writeFn(unsafe.Pointer(&req[0]), int32(len(req)))
	if n != int32(len(req)) {
		return ModbusError(fmt.Sprintf("串口写入失败: wrote %d/%d bytes", n, len(req)))
	}

	// 等待响应
	m.sleepFn(m.delayMs)

	// 读取响应
	resp := make([]byte, 64)
	n = m.readFn(unsafe.Pointer(&resp[0]), int32(len(resp)))
	if n < 5 {
		return ModbusError(fmt.Sprintf("响应数据不完整，收到 %d 字节", n))
	}

	// 验证 CRC
	if !CheckCRC(resp[:n]) {
		return ModbusError("CRC 校验失败")
	}

	// 验证响应
	if n < 8 {
		return ModbusError("响应数据长度不足")
	}
	if resp[0] != m.addr {
		return ModbusError("设备地址不匹配")
	}
	if resp[1] != 0x06 {
		return ModbusError("功能码不匹配")
	}

	return nil
}

// CheckCRC 验证 CRC 校验码
func CheckCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	crc := CRC16(data[:len(data)-2])
	rcvCrc := uint16(data[len(data)-2]) | uint16(data[len(data)-1])<<8
	return crc == rcvCrc
}
