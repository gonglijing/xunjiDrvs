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

func CheckCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	got := uint16(data[len(data)-2]) | uint16(data[len(data)-1])<<8
	return CRC16(data[:len(data)-2]) == got
}
