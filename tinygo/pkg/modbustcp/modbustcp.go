// Package modbustcp 提供 TinyGo 驱动共享的 Modbus TCP 读协议工具。
//
// 与 RTU 版本相比，TCP 侧没有 CRC，但多了 MBAP 头。
// 这里同样只抽取最稳定的重复部分，让驱动文件聚焦设备点位映射，
// 而不是反复重写请求头拼接和响应拆包逻辑。
package modbustcp

import "encoding/binary"

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errf(s string) error { return simpleErr(s) }

type TransceiveFunc func(req []byte, resp []byte, timeoutMs int) int

// BuildReadRequest 组装标准的 Modbus TCP 读保持寄存器请求。
//
// 目前驱动里事务号固定为 0x0001，协议标识固定为 0x0000。
// 这对当前“单次请求-单次响应”的简单驱动场景已经足够，也最容易读懂。
func BuildReadRequest(addr byte, funcCode byte, startReg uint16, count uint16) []byte {
	mbap := make([]byte, 12)
	mbap[0] = 0x00
	mbap[1] = 0x01
	mbap[2] = 0x00
	mbap[3] = 0x00
	mbap[4] = 0x00
	mbap[5] = 0x06
	mbap[6] = addr
	mbap[7] = funcCode
	mbap[8] = byte(startReg >> 8)
	mbap[9] = byte(startReg)
	mbap[10] = byte(count >> 8)
	mbap[11] = byte(count)
	return mbap
}

// ParseReadResponse 解析宿主返回的 Modbus TCP 响应。
//
// 这里默认调用方已经拿到一个完整的 TCP 响应帧，因此只做：
// - 基本长度检查
// - 地址和功能码匹配检查
// - 字节数检查
// - 寄存器拆包
func ParseReadResponse(data []byte, addr byte, funcCode byte) ([]uint16, error) {
	if len(data) < 9 {
		return nil, errf("响应数据不完整")
	}
	pdu := data[6:]
	if len(pdu) < 3 {
		return nil, errf("响应数据不完整")
	}
	if pdu[0] != addr || pdu[1] != funcCode {
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

// ReadRegisters 提供标准的 Modbus TCP 寄存器读取流程。
//
// 它与 RTU 版本的设计目标相同：把最常见的通信样板收敛到共享包，
// 同时把“重试策略”“设备特殊兼容逻辑”留给具体驱动自己处理。
func ReadRegisters(
	transceive TransceiveFunc,
	addr byte,
	funcCode byte,
	startReg uint16,
	count uint16,
	timeoutMs int,
	respSize int,
) ([]uint16, error) {
	req := BuildReadRequest(addr, funcCode, startReg, count)
	resp := make([]byte, respSize)

	n := transceive(req, resp, timeoutMs)
	if n < 9 {
		return nil, errf("响应数据不完整")
	}

	values, err := ParseReadResponse(resp[:n], addr, funcCode)
	if err != nil {
		return nil, err
	}
	if len(values) < int(count) {
		return nil, errf("寄存器数量不足")
	}
	return values, nil
}
