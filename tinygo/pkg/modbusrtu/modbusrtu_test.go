package modbusrtu

import "testing"

func TestBuildWriteSingleFrame(t *testing.T) {
	frame := BuildWriteSingleFrame(0x01, 0x06, 0x0010, 0x1234)
	want := []byte{0x01, 0x06, 0x00, 0x10, 0x12, 0x34, 0x84, 0xBD}
	if len(frame) != len(want) {
		t.Fatalf("frame len = %d, want %d", len(frame), len(want))
	}
	for i := range want {
		if frame[i] != want[i] {
			t.Fatalf("frame[%d] = 0x%02X, want 0x%02X", i, frame[i], want[i])
		}
	}
}

func TestParseWriteSingleResponse(t *testing.T) {
	resp := []byte{0x01, 0x06, 0x00, 0x10, 0x12, 0x34, 0x84, 0xBD}
	reg, value, err := ParseWriteSingleResponse(resp, 0x01, 0x06)
	if err != nil {
		t.Fatalf("ParseWriteSingleResponse error = %v", err)
	}
	if reg != 0x0010 {
		t.Fatalf("reg = 0x%04X, want 0x0010", reg)
	}
	if value != 0x1234 {
		t.Fatalf("value = 0x%04X, want 0x1234", value)
	}
}

func TestWriteSingleRegister(t *testing.T) {
	transceive := func(req []byte, respLen int, timeoutMs int) ([]byte, int) {
		if respLen != 8 {
			t.Fatalf("respLen = %d, want 8", respLen)
		}
		if timeoutMs != 1000 {
			t.Fatalf("timeoutMs = %d, want 1000", timeoutMs)
		}
		resp := append([]byte(nil), req...)
		return resp, len(resp)
	}

	reg, value, err := WriteSingleRegister(transceive, 0x01, 0x06, 0x0010, 0x1234, 1000, false, 16, nil)
	if err != nil {
		t.Fatalf("WriteSingleRegister error = %v", err)
	}
	if reg != 0x0010 || value != 0x1234 {
		t.Fatalf("write echo = (0x%04X, 0x%04X), want (0x0010, 0x1234)", reg, value)
	}
}
