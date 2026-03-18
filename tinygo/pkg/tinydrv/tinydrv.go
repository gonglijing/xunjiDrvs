package tinydrv

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	pdk "github.com/extism/go-pdk"
)

type InvocationEnvelope struct {
	Config map[string]string `json:"config"`
}

type Point struct {
	FieldName string `json:"field_name"`
	Value     string `json:"value"`
	RW        string `json:"rw"`
	Unit      string `json:"unit"`
	Label     string `json:"label"`
}

type HandleResponse struct {
	Success    bool    `json:"success"`
	ProductKey string  `json:"productKey"`
	Points     []Point `json:"points"`
	Error      string  `json:"error,omitempty"`
}

type DescribeResponse struct {
	Success bool     `json:"success"`
	Data    struct{} `json:"data"`
}

type VersionData struct {
	Version    string `json:"version"`
	ProductKey string `json:"productKey"`
}

type VersionResponse struct {
	Success bool        `json:"success"`
	Data    VersionData `json:"data"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func ParseConfigMap() map[string]string {
	var envelope InvocationEnvelope
	if err := pdk.InputJSON(&envelope); err != nil {
		return nil
	}
	return envelope.Config
}

func ParseString(config map[string]string, key string, def string) string {
	if config == nil {
		return def
	}
	if value := strings.TrimSpace(config[key]); value != "" {
		return value
	}
	return def
}

func ParseInt(config map[string]string, key string, def int) int {
	value := ParseString(config, key, "")
	if value == "" {
		return def
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return def
	}
	return n
}

func ParseBool(config map[string]string, key string, def bool) bool {
	value := ParseString(config, key, "")
	if value == "" {
		return def
	}
	return value == "1" || strings.EqualFold(value, "true")
}

func FormatFloat(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

func OutputJSON(v interface{}) {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		b = []byte(`{"success":false,"error":"encode failed"}`)
	}
	pdk.Output(b)
}

func Logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	pdk.Log(pdk.LogDebug, msg)
}

func HexPreview(b []byte, n int, max int) string {
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
