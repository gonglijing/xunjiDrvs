// =============================================================================
// 温湿度传感器 Modbus RTU 驱动
// =============================================================================
// 本驱动用于温湿度传感器，寄存器配置硬编码在驱动内部。
// 网关侧只需提供串口资源ID和设备地址。
//
// 网关侧配置 (ConfigSchema):
// {
//     "resource_id": 1,          // 串口资源ID
//     "device_address": 1        // Modbus 设备地址
// }
//
// 编译命令 (TinyGo):
//   tinygo build -o temperature_humidity.wasm -target=wasi temperature_humidity.go
//
// Host Functions 需要 (网关提供):
//   - serial_read: 从串口读取数据
//   - serial_write: 向串口写入数据
//   - sleep_ms: 毫秒延时
//   - output: 输出日志
// =============================================================================

package main

import (
	"fmt"
	"unsafe"

	"github.com/gonglijing/xunjiFsu/drvs/modbus"
)

// =============================================================================
// Host Functions 接口声明
// =============================================================================

//go:export serial_read
func serial_read(buf unsafe.Pointer, size int32) int32

//go:export serial_write
func serial_write(buf unsafe.Pointer, size int32) int32

//go:export sleep_ms
func sleep_ms(ms int32)

//go:export output
func output(ptr unsafe.Pointer, size int32)

// Wrapper functions for modbus RTU callbacks (can't use //go:export functions directly as values)
func serialReadWrapper(buf unsafe.Pointer, size int32) int32 {
	return serial_read(buf, size)
}

func serialWriteWrapper(buf unsafe.Pointer, size int32) int32 {
	return serial_write(buf, size)
}

func sleepMsWrapper(ms int32) {
	sleep_ms(ms)
}

// =============================================================================
// 驱动主入口
// =============================================================================

// registerDef 寄存器定义 (驱动定义设备规格)
type registerDef struct {
	addr  uint16  // 寄存器地址
	name  string  // 测点名称
	vtype string  // 数据类型
	scale float64 // 缩放因子
	rw    string  // 读写特征: "R" (只读) | "W" (只写) | "RW" (读写)
}

// registers 本驱动支持的寄存器映射 (设备规格由驱动定义)
var registers = []registerDef{
	{addr: 0, name: "temperature", vtype: "int16", scale: 0.1, rw: "R"},     // 温度 → 寄存器 0, 只读
	{addr: 1, name: "humidity", vtype: "int16", scale: 0.1, rw: "R"},        // 湿度 → 寄存器 1, 只读
	{addr: 0x07D0, name: "device_addr", vtype: "uint16", scale: 1, rw: "W"}, // 地址 → 0x07D0, 只写
}

//go:export collect
func collect() {
	config := getConfig()

	// 创建 Modbus RTU 客户端
	modbusRTU := modbus.NewModbusRTU(
		byte(config.DeviceAddress),
		serialReadWrapper,
		serialWriteWrapper,
		sleepMsWrapper,
		150,
	)

	switch config.FuncName {
	case "read":
		// 驱动决定读什么 - 读取所有定义的寄存器
		points := make([]map[string]interface{}, 0)
		errors := []string{}

		for _, reg := range registers {
			value, err := modbusRTU.ReadHoldingRegister(reg.addr)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", reg.name, err))
				continue
			}
			// 数据类型转换和缩放
			actualValue := parseValue(value, reg.vtype, reg.scale)
			// 返回测点名、实时值、读写特征
			points = append(points, map[string]interface{}{
				"field_name": reg.name,
				"value":      actualValue,
				"rw":         reg.rw,
			})
		}

		if len(errors) > 0 {
			outputString(fmt.Sprintf(`{"success":false,"error":"%s"}`, joinErrors(errors)))
			return
		}
		outputString(fmt.Sprintf(`{"success":true,"data":{"points":%s}}`, sliceToJSON(points)))

	case "write":
		// 网关决定写什么 (field_name + value)，驱动决定怎么写 (寄存器地址)
		reg := findRegister(config.FieldName)
		if reg.addr == 0 && config.FieldName != "device_addr" {
			outputString(fmt.Sprintf(`{"success":false,"error":"未知测点: %s"}`, config.FieldName))
			return
		}
		// 转换 value 为 uint16
		writeValue := uint16(config.Value)
		err := modbusRTU.WriteHoldingRegister(reg.addr, writeValue)
		if err != nil {
			outputString(fmt.Sprintf(`{"success":false,"error":"%v"}`, err))
			return
		}
		outputString(fmt.Sprintf(`{"success":true,"message":"%s = %d"}`, config.FieldName, writeValue))

	default:
		outputString(`{"success":false,"error":"未知操作"}`)
	}
}

// findRegister 根据测点名查找寄存器定义
func findRegister(fieldName string) registerDef {
	for _, reg := range registers {
		if reg.name == fieldName {
			return reg
		}
	}
	return registerDef{}
}

// =============================================================================
// Modbus 协议操作 (使用 modbus 包)
// =============================================================================

// GatewayConfig 网关传递的配置
type GatewayConfig struct {
	ResourceID    int64   `json:"resource_id"`    // 串口资源ID
	DeviceAddress int     `json:"device_address"` // Modbus 设备地址
	FuncName      string  `json:"func_name"`      // 功能名: "read" | "write"
	FieldName     string  `json:"field_name"`     // 测点名 (用于 write, 网关决定)
	Value         float64 `json:"value"`          // 值 (用于 write, 网关决定)
}

// getConfig 从 Extism 配置获取网关配置
func getConfig() GatewayConfig {
	return GatewayConfig{
		ResourceID:    1,
		DeviceAddress: 1,
		FuncName:      "read",
		FieldName:     "",
		Value:         0,
	}
}

// parseValue 根据类型和缩放因子解析原始值
func parseValue(raw uint16, dataType string, scale float64) float64 {
	switch dataType {
	case "int16":
		return modbus.Int16ToFloat64(raw, scale)
	case "uint16":
		return modbus.Uint16ToFloat64(raw, scale)
	default:
		return modbus.Int16ToFloat64(raw, scale)
	}
}

// joinErrors 合并错误消息
func joinErrors(errors []string) string {
	result := ""
	for i, e := range errors {
		if i > 0 {
			result += "; "
		}
		result += e
	}
	return result
}

// mapToJSON 将 map 转为 JSON 字符串
func mapToJSON(data map[string]float64) string {
	result := "{"
	first := true
	for k, v := range data {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":%.1f`, k, v)
		first = false
	}
	result += "}"
	return result
}

// sliceToJSON 将 []map[string]interface{} 转为 JSON 字符串
func sliceToJSON(points []map[string]interface{}) string {
	result := "["
	for i, p := range points {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`{"field_name":"%s","value":%.1f,"rw":"%s"}`,
			p["field_name"], p["value"], p["rw"])
	}
	result += "]"
	return result
}

// =============================================================================
// 辅助函数
// =============================================================================

func outputString(s string) {
	// TinyGo WASI compatible pointer conversion
	ptr := unsafe.Pointer(unsafe.StringData(s))
	output(ptr, int32(len(s)))
}

// =============================================================================

func main() {}
