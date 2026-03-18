// Package modbusrtu 提供 TinyGo 驱动共享的 Modbus RTU 读协议工具。
//
// 这个包刻意只覆盖“最稳定、最重复”的那一小层：
// - 读寄存器请求帧组装
// - 响应帧校验与解析
// - CRC16 计算
// - 带可选调试日志的标准读流程
//
// 这样做的目标不是把所有 RTU 驱动抽象成一套复杂框架，而是让各个驱动文件
// 把注意力集中在“读哪些寄存器、如何把寄存器转成业务点位”上。
package modbusrtu

import (
	"strconv"

	"github.com/gonglijing/xunjiFsu/drvs/tinygo/pkg/tinydrv"
)

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errf(s string) error { return simpleErr(s) }

type TransceiveFunc func(req []byte, respLen int, timeoutMs int) ([]byte, int)

type LoggerFunc func(format string, args ...interface{})

// BuildReadFrame 构造标准的 Modbus RTU 读寄存器请求帧。
//
// 当前驱动体系里最常见的是功能码 0x03/0x04，因此这里直接聚焦读取场景。
// 写寄存器、批量写等更少见的行为暂时不下沉，以免公共层抽象过早膨胀。
func BuildReadFrame(addr byte, funcCode byte, start uint16, qty uint16) []byte {
	req := make([]byte, 8)
	req[0] = addr
	req[1] = funcCode
	req[2], req[3] = byte(start>>8), byte(start)
	req[4], req[5] = byte(qty>>8), byte(qty)
	crc := CRC16(req[:6])
	req[6], req[7] = byte(crc), byte(crc>>8)
	return req
}

// BuildWriteSingleFrame 构造标准的 Modbus RTU 单寄存器写请求帧。
//
// 当前先下沉最常见的 0x06 写单寄存器场景。
// 对大多数“远程开关、模式切换、设定值写入”类设备来说，这已经覆盖最常用的主动控制能力。
func BuildWriteSingleFrame(addr byte, funcCode byte, reg uint16, value uint16) []byte {
	req := make([]byte, 8)
	req[0] = addr
	req[1] = funcCode
	req[2], req[3] = byte(reg>>8), byte(reg)
	req[4], req[5] = byte(value>>8), byte(value)
	crc := CRC16(req[:6])
	req[6], req[7] = byte(crc), byte(crc>>8)
	return req
}

// ParseReadResponse 解析一个 Modbus RTU 读响应。
//
// 这里把常见失败路径尽量区分开：
// - 地址不匹配
// - 设备返回异常码
// - 功能码不匹配
// - 字节数不匹配
// - CRC 校验失败
//
// 这样驱动层在 debug 模式下能拿到更可读的错误信息，而不是统一的“读失败”。
func ParseReadResponse(data []byte, addr byte, funcCode byte) ([]uint16, error) {
	if len(data) < 5 || data[0] != addr {
		return nil, errf("invalid response")
	}
	if len(data) >= 3 && data[1] == (funcCode|0x80) {
		return nil, errf("modbus exception code=" + strconv.Itoa(int(data[2])))
	}
	if data[1] != funcCode {
		return nil, errf("unexpected function code")
	}
	byteCnt := int(data[2])
	if byteCnt < 2 || len(data) < 3+byteCnt+2 {
		return nil, errf("byte count mismatch")
	}
	if !CheckCRC(data[:3+byteCnt+2]) {
		return nil, errf("crc error")
	}

	values := make([]uint16, byteCnt/2)
	for i := 0; i < len(values); i++ {
		values[i] = uint16(data[3+i*2])<<8 | uint16(data[4+i*2])
	}
	return values, nil
}

// ParseWriteSingleResponse 解析 Modbus RTU 单寄存器写响应。
//
// 0x06 的正常响应会把“寄存器地址 + 写入值”原样回显回来，
// 因此这里除了做基础帧校验外，还会把回显内容返回给调用方，方便上层做二次确认或日志记录。
func ParseWriteSingleResponse(data []byte, addr byte, funcCode byte) (uint16, uint16, error) {
	if len(data) < 8 || data[0] != addr {
		return 0, 0, errf("invalid response")
	}
	if len(data) >= 3 && data[1] == (funcCode|0x80) {
		return 0, 0, errf("modbus exception code=" + strconv.Itoa(int(data[2])))
	}
	if data[1] != funcCode {
		return 0, 0, errf("unexpected function code")
	}
	if len(data) < 8 {
		return 0, 0, errf("response too short")
	}
	if !CheckCRC(data[:8]) {
		return 0, 0, errf("crc error")
	}

	reg := uint16(data[2])<<8 | uint16(data[3])
	value := uint16(data[4])<<8 | uint16(data[5])
	return reg, value, nil
}

// ReadRegisters 封装 TinyGo 驱动里最常见的一段 RTU 读流程。
//
// 它串起以下步骤：
// 1. 组装请求帧。
// 2. 调用具体驱动提供的 transceive 函数与宿主通信。
// 3. 在 debug 模式下输出请求/响应预览。
// 4. 校验并解析响应。
// 5. 检查寄存器数量是否满足调用方预期。
//
// 这个函数刻意不吞掉错误，也不在内部做重试或地址回退。
// 是否回退到 0-based 地址、是否切片重试，应该由具体驱动决定，
// 因为那已经属于“设备兼容策略”，不是通用 RTU 读流程本身。
func ReadRegisters(
	transceive TransceiveFunc,
	addr byte,
	funcCode byte,
	startReg uint16,
	count uint16,
	timeoutMs int,
	debug bool,
	previewMax int,
	logf LoggerFunc,
) ([]uint16, error) {
	req := BuildReadFrame(addr, funcCode, startReg, count)
	if debug && logf != nil {
		logf("rtu req fc=%02X % X", funcCode, req)
	}

	resp, n := transceive(req, int(count)*2+5, timeoutMs)
	if debug && logf != nil {
		logf("rtu fc=%02X n=%d resp=%s", funcCode, n, tinydrv.HexPreview(resp, n, previewMax))
	}
	if n <= 0 {
		return nil, errf("read timeout")
	}

	values, err := ParseReadResponse(resp[:n], addr, funcCode)
	if err != nil {
		if debug && logf != nil {
			logf("parse err=%v", err)
		}
		return nil, err
	}
	if len(values) < int(count) {
		err := errf("insufficient register data")
		if debug && logf != nil {
			logf("parse err=%v", err)
		}
		return nil, err
	}
	return values, nil
}

// WriteSingleRegister 封装标准的 Modbus RTU 单寄存器写流程。
//
// 与 ReadRegisters 一样，这里只负责“标准协议往返”本身：
// 1. 组装 0x06 请求帧
// 2. 通过宿主收发
// 3. 在 debug 模式下输出请求/响应预览
// 4. 校验响应并返回回显的地址和值
//
// 是否允许写某个字段、字段值如何从业务值映射到寄存器值，应由具体驱动自己决定。
func WriteSingleRegister(
	transceive TransceiveFunc,
	addr byte,
	funcCode byte,
	reg uint16,
	value uint16,
	timeoutMs int,
	debug bool,
	previewMax int,
	logf LoggerFunc,
) (uint16, uint16, error) {
	req := BuildWriteSingleFrame(addr, funcCode, reg, value)
	if debug && logf != nil {
		logf("rtu write req fc=%02X % X", funcCode, req)
	}

	resp, n := transceive(req, 8, timeoutMs)
	if debug && logf != nil {
		logf("rtu write fc=%02X n=%d resp=%s", funcCode, n, tinydrv.HexPreview(resp, n, previewMax))
	}
	if n <= 0 {
		return 0, 0, errf("write timeout")
	}

	writtenReg, writtenValue, err := ParseWriteSingleResponse(resp[:n], addr, funcCode)
	if err != nil {
		if debug && logf != nil {
			logf("write parse err=%v", err)
		}
		return 0, 0, err
	}
	return writtenReg, writtenValue, nil
}

// CRC16 计算 Modbus RTU 使用的 CRC-16/Modbus 校验值。
//
// 该实现使用最直观的逐字节、逐位算法。虽然表驱动实现更快，
// 但这里的调用频率和数据量都很小，可读性比极限性能更重要。
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

// CheckCRC 仅负责校验一整帧末尾携带的 CRC 是否正确。
// 调用方需要确保传入的 data 至少包含 2 个 CRC 字节。
func CheckCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	got := uint16(data[len(data)-2]) | uint16(data[len(data)-1])<<8
	return CRC16(data[:len(data)-2]) == got
}
