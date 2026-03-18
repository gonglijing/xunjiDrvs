package modbustcp

import "testing"

func TestBuildWriteSingleRequest(t *testing.T) {
	req := BuildWriteSingleRequest(0x01, 0x06, 0x0010, 0x1234)
	want := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x06, 0x00, 0x10, 0x12, 0x34}
	if len(req) != len(want) {
		t.Fatalf("req len = %d, want %d", len(req), len(want))
	}
	for i := range want {
		if req[i] != want[i] {
			t.Fatalf("req[%d] = 0x%02X, want 0x%02X", i, req[i], want[i])
		}
	}
}

func TestParseWriteSingleResponse(t *testing.T) {
	resp := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x06, 0x00, 0x10, 0x12, 0x34}
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
	transceive := func(req []byte, resp []byte, timeoutMs int) int {
		if timeoutMs != 1000 {
			t.Fatalf("timeoutMs = %d, want 1000", timeoutMs)
		}
		copy(resp, req)
		return len(req)
	}

	reg, value, err := WriteSingleRegister(transceive, 0x01, 0x06, 0x0010, 0x1234, 1000, 16)
	if err != nil {
		t.Fatalf("WriteSingleRegister error = %v", err)
	}
	if reg != 0x0010 || value != 0x1234 {
		t.Fatalf("write echo = (0x%04X, 0x%04X), want (0x0010, 0x1234)", reg, value)
	}
}
