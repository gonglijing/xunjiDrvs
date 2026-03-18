package modbustcp

import "encoding/binary"

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errf(s string) error { return simpleErr(s) }

type TransceiveFunc func(req []byte, resp []byte, timeoutMs int) int

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
