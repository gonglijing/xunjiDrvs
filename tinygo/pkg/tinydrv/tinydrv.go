// Package tinydrv 提供 TinyGo 驱动共享的轻量级基础设施。
//
// 这些辅助函数看起来都很小，但它们的作用是把“插件框架层”的重复代码
// 从具体驱动里挪走，让每个驱动文件更接近“协议说明书 + 点位映射表”。
//
// 当前包主要负责：
// - 解析网关传入的 config 映射
// - 输出统一 JSON 响应
// - 提供公共响应结构
// - 提供调试日志与十六进制预览
// - 提供一些最常见的字符串/数字格式化函数
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

// Point 对应网关侧约定的单个测点输出格式。
// 几乎所有只读驱动都会把寄存器值最终映射为这个结构。
type Point struct {
	FieldName string `json:"field_name"`
	Value     string `json:"value"`
	RW        string `json:"rw"`
	Unit      string `json:"unit"`
	Label     string `json:"label"`
}

// HandleResponse 是 handle 导出函数最常见的成功/失败返回结构。
// 单独放在公共包里，可以避免每个驱动文件重复声明一遍完全相同的结构体。
type HandleResponse struct {
	Success    bool    `json:"success"`
	ProductKey string  `json:"productKey"`
	Points     []Point `json:"points"`
	Error      string  `json:"error,omitempty"`
}

// DescribeResponse 对应 describe 导出函数的统一返回格式。
// 当前大多数驱动没有可写点位，因此 data 常常是空对象。
type DescribeResponse struct {
	Success bool     `json:"success"`
	Data    struct{} `json:"data"`
}

// VersionData/VersionResponse 对应 version 导出函数的固定输出。
// 把这两个结构放在公共包里后，驱动文件只需要填自己的 version 与 productKey。
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

// NewHandleSuccess 统一构造 handle 成功返回。
//
// 约定上 handle 无论哪个驱动，外层都应保持 success/productKey/points/error 这组字段。
// 成功时 points 至少返回空数组而不是 null，便于宿主和前端按固定结构消费。
func NewHandleSuccess(productKey string, points []Point) HandleResponse {
	if points == nil {
		points = make([]Point, 0)
	}
	return HandleResponse{
		Success:    true,
		ProductKey: strings.TrimSpace(productKey),
		Points:     points,
	}
}

// NewHandleError 统一构造 handle 失败返回。
//
// 这里故意不再单独走只有 success/error 的极简结构，
// 而是保持和成功返回相同的外层字段，减少宿主侧分支判断。
func NewHandleError(productKey string, errText string) HandleResponse {
	return HandleResponse{
		Success:    false,
		ProductKey: strings.TrimSpace(productKey),
		Points:     make([]Point, 0),
		Error:      strings.TrimSpace(errText),
	}
}

func OutputHandleSuccess(productKey string, points []Point) {
	OutputJSON(NewHandleSuccess(productKey, points))
}

func OutputHandleError(productKey string, errText string) {
	OutputJSON(NewHandleError(productKey, errText))
}

// ParseConfigMap 解析 Extism 输入中的 config 映射。
//
// 网关侧通常会把插件入参包装为一个更大的 JSON 对象，而驱动实际最常关心的
// 只有 config。这里直接把外层 envelope 拆掉，减少每个驱动里的样板代码。
func ParseConfigMap() map[string]string {
	var envelope InvocationEnvelope
	if err := pdk.InputJSON(&envelope); err != nil {
		return nil
	}
	return envelope.Config
}

// ParseString/ParseInt/ParseBool 提供“带默认值”的配置解析。
//
// 这些函数统一处理了几个重复细节：
// - config 为空
// - 字符串需要 trim
// - 解析失败时回退默认值
//
// 这样驱动层就不需要每次都写一长串 if/trim/parse/error fallback。
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

// IsWriteFunc 统一判断一次驱动调用是否是写操作。
//
// 大部分驱动只需要区分“正常采集(read/collect)”和“下行写(write)”两条路径，
// 这里把 trim + ignore-case 的细节收敛起来，避免每个驱动重复实现。
func IsWriteFunc(funcName string) bool {
	return strings.EqualFold(strings.TrimSpace(funcName), "write")
}

// FormatFloat 统一格式化驱动输出中的数值字符串。
// 单独收敛到公共包后，驱动文件里不必再重复一行 strconv.FormatFloat。
func FormatFloat(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

// OutputJSON 负责把任意响应结构输出给 Extism 宿主。
//
// 这里保留了一个非常保守的兜底行为：如果编码失败，至少输出一段固定错误 JSON，
// 避免宿主侧拿到空输出后更难定位问题。
func OutputJSON(v interface{}) {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		b = []byte(`{"success":false,"error":"encode failed"}`)
	}
	pdk.Output(b)
}

// Logf 是驱动统一使用的 debug 日志入口。
// 之所以保持 printf 风格，是因为 Modbus 调试天然适合拼接寄存器地址、长度和十六进制预览。
func Logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	pdk.Log(pdk.LogDebug, msg)
}

// HexPreview 把一段响应裁剪成适合日志输出的十六进制字符串。
//
// max 的存在是为了避免把整帧原样刷进日志，尤其是点位较多、响应较长时。
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
