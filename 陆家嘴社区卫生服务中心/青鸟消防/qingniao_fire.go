// =============================================================================
// 青鸟消防主机 - Modbus RTU 驱动（陆家嘴社区卫生服务中心）
// =============================================================================
//
// 协议来源:
//   - 数据库: /Users/mac/workspace/pandav2/gateway/deploy/pandax230.db
//   - 设备: 青鸟消防主机 (did=t8AyvifdGo, protocol=modbus-rtu)
//
// 点表摘要:
//   - 总点位: 275
//   - 功能码: 0x03 (HOLDING_REGISTER)
//   - 原始连续地址段: 257~416, 513~627
//   - 读取分片(每次<=50寄存器): 257+50, 307+50, 357+50, 407+10, 513+50, 563+50, 613+15
//
// Host 提供: serial_transceive
//
// =============================================================================
package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	pdk "github.com/extism/go-pdk"
)

//go:wasmimport extism:host/user serial_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

type DriverConfig struct {
	DeviceAddress int    `json:"device_address"`
	FuncName      string `json:"func_name"`
	FieldName     string `json:"field_name"`
	Value         string `json:"value"`
	Debug         bool   `json:"debug"`
}

const DriverVersion = "1.0.0"

const (
	FUNC_CODE_READ_HOLDING = 0x03
	FUNC_CODE_READ_INPUT   = 0x04
)

type PointConfig struct {
	Field    string
	Address  uint16
	Scale    float64
	Decimals int
	RW       string
	Unit     string
	Label    string
}

var pointConfig = []PointConfig{
	{Field: "yg0101", Address: 257, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层心电烟感"},
	{Field: "yg0102", Address: 258, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层心电烟感"},
	{Field: "yg0103", Address: 259, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层外科烟感"},
	{Field: "yg0104", Address: 260, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层外科烟感"},
	{Field: "yg0105", Address: 261, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层外科烟感"},
	{Field: "yg0106", Address: 262, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层B超烟感"},
	{Field: "yg0107", Address: 263, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层楼梯口烟感"},
	{Field: "yg0108", Address: 264, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层楼梯口烟感"},
	{Field: "yg0109", Address: 265, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层化验科烟感"},
	{Field: "yg010a", Address: 266, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层化验科烟感"},
	{Field: "yg010b", Address: 267, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层化验科烟感"},
	{Field: "yg010c", Address: 268, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层化验科烟感"},
	{Field: "yg010d", Address: 269, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层化验科烟感"},
	{Field: "yg010e", Address: 270, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层化验科烟感"},
	{Field: "yg010f", Address: 271, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层化验科烟感"},
	{Field: "yg0110", Address: 272, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层全科诊室烟感"},
	{Field: "yg0111", Address: 273, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层全科诊室烟感"},
	{Field: "yg0112", Address: 274, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层全科诊室烟感"},
	{Field: "yg0113", Address: 275, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层全科诊室烟感"},
	{Field: "yg0114", Address: 276, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层全科诊室烟感"},
	{Field: "yg0115", Address: 277, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层全科诊室烟感"},
	{Field: "yg0116", Address: 278, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层全科诊室烟感"},
	{Field: "yg0117", Address: 279, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层大门口烟感"},
	{Field: "yg0118", Address: 280, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层更衣室烟感"},
	{Field: "yg0119", Address: 281, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层收费处烟感"},
	{Field: "yg011a", Address: 282, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层五官科烟感"},
	{Field: "yg011b", Address: 283, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层五官科烟感"},
	{Field: "yg011c", Address: 284, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层五官科烟感"},
	{Field: "yg011d", Address: 285, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层垃圾房烟感"},
	{Field: "yg011e", Address: 286, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层补液大厅烟感"},
	{Field: "yg011f", Address: 287, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层注射室烟感"},
	{Field: "yg0120", Address: 288, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层配电室烟感"},
	{Field: "yg0121", Address: 289, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层补液大厅烟感"},
	{Field: "yg0122", Address: 290, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层补液大厅烟感"},
	{Field: "yg0123", Address: 291, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层抢救室烟感"},
	{Field: "yg0124", Address: 292, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层配电室烟感"},
	{Field: "yg0125", Address: 293, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg0126", Address: 294, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg0127", Address: 295, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层药库烟感"},
	{Field: "yg0128", Address: 296, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层药库烟感"},
	{Field: "yg0129", Address: 297, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层药库烟感"},
	{Field: "yg012a", Address: 298, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg012b", Address: 299, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg012c", Address: 300, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg012d", Address: 301, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg012e", Address: 302, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg012f", Address: 303, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg0130", Address: 304, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "yg0131", Address: 305, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道烟感"},
	{Field: "sb0132", Address: 306, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道手报"},
	{Field: "sb0133", Address: 307, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道手报"},
	{Field: "sb0134", Address: 308, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道手报"},
	{Field: "xb0135", Address: 309, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道消报"},
	{Field: "xb0136", Address: 310, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道消报"},
	{Field: "xb0137", Address: 311, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层消报"},
	{Field: "xb0138", Address: 312, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层消报"},
	{Field: "slzs0139", Address: 313, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层水流指示"},
	{Field: "xhf013a", Address: 314, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层信号阀"},
	{Field: "sg013b", Address: 315, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层声光"},
	{Field: "sg013c", Address: 316, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层声光"},
	{Field: "sg013d", Address: 317, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层声光"},
	{Field: "yg013e", Address: 318, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医诊室烟感"},
	{Field: "yg013f", Address: 319, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医诊室烟感"},
	{Field: "yg0140", Address: 320, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医诊室烟感"},
	{Field: "yg0141", Address: 321, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层楼梯口烟感"},
	{Field: "yg0142", Address: 322, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医诊室烟感"},
	{Field: "yg0143", Address: 323, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医诊室烟感"},
	{Field: "yg0144", Address: 324, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医诊室烟感"},
	{Field: "yg0145", Address: 325, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医诊室烟感"},
	{Field: "yg0146", Address: 326, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层口腔科烟感"},
	{Field: "yg0147", Address: 327, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层口腔科烟感"},
	{Field: "yg0148", Address: 328, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层口腔科烟感"},
	{Field: "yg0149", Address: 329, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层口腔科烟感"},
	{Field: "yg014a", Address: 330, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层机房烟感"},
	{Field: "yg014b", Address: 331, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层楼梯口烟感"},
	{Field: "yg014c", Address: 332, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层康复诊疗室烟感"},
	{Field: "yg014d", Address: 333, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层康复诊疗室烟感"},
	{Field: "yg014e", Address: 334, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层全科诊室烟感"},
	{Field: "yg014f", Address: 335, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层心电图室烟感"},
	{Field: "yg0150", Address: 336, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层医生办公室烟感"},
	{Field: "yg0151", Address: 337, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层医生办公室烟感"},
	{Field: "yg0152", Address: 338, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层医生办公室烟感"},
	{Field: "yg0153", Address: 339, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中药房烟感"},
	{Field: "yg0154", Address: 340, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层收费烟感"},
	{Field: "yg0155", Address: 341, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层楼梯口烟感"},
	{Field: "yg0156", Address: 342, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道烟感"},
	{Field: "yg0157", Address: 343, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道烟感"},
	{Field: "yg0158", Address: 344, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道烟感"},
	{Field: "yg0159", Address: 345, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道烟感"},
	{Field: "wg015a", Address: 346, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医便诊室温感"},
	{Field: "wg015b", Address: 347, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医便诊室温感"},
	{Field: "wg015c", Address: 348, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医便诊室温感"},
	{Field: "wg015d", Address: 349, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医便诊室温感"},
	{Field: "wg015e", Address: 350, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医便诊室温感"},
	{Field: "wg015f", Address: 351, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医便诊室温感"},
	{Field: "wg0160", Address: 352, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层中医便诊室温感"},
	{Field: "sb0161", Address: 353, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道手报"},
	{Field: "sb0162", Address: 354, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道手报"},
	{Field: "sb0163", Address: 355, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道手报"},
	{Field: "xb0164", Address: 356, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道消报"},
	{Field: "xb0165", Address: 357, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道消报"},
	{Field: "xb0166", Address: 358, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道消报"},
	{Field: "xb0167", Address: 359, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道消报"},
	{Field: "xhf0168", Address: 360, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层信号阀"},
	{Field: "lszs0169", Address: 361, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层水流指示"},
	{Field: "sg016a", Address: 362, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层声光"},
	{Field: "sg016b", Address: 363, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层声光"},
	{Field: "sg016c", Address: 364, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层声光"},
	{Field: "yg016d", Address: 365, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层杂物间烟感"},
	{Field: "yg016e", Address: 366, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层档案室烟感"},
	{Field: "yg016f", Address: 367, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层档案室烟感"},
	{Field: "yg0170", Address: 368, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层档案室烟感"},
	{Field: "yg0171", Address: 369, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层杂物间烟感"},
	{Field: "yg0172", Address: 370, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层餐厅烟感"},
	{Field: "yg0173", Address: 371, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层餐厅烟感"},
	{Field: "yg0174", Address: 372, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层餐厅烟感"},
	{Field: "yg0175", Address: 373, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层摄片机房烟感"},
	{Field: "yg0176", Address: 374, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层摄片机房烟感"},
	{Field: "yg0177", Address: 375, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层摄片机房烟感"},
	{Field: "yg0178", Address: 376, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层机房控制室烟感"},
	{Field: "yg0179", Address: 377, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层候诊室烟感"},
	{Field: "yg017a", Address: 378, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层登记室烟感"},
	{Field: "yg017b", Address: 379, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层机房控制室烟感"},
	{Field: "yg017c", Address: 380, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层透视机房烟感"},
	{Field: "wg017d", Address: 381, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层厨房温感"},
	{Field: "wg017e", Address: 382, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层厨房温感"},
	{Field: "wg017f", Address: 383, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层厨房温感"},
	{Field: "xb0180", Address: 384, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层候诊室消报"},
	{Field: "xb0181", Address: 385, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层候诊室消报"},
	{Field: "xb0182", Address: 386, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道消报"},
	{Field: "xb0183", Address: 387, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道消报"},
	{Field: "xb0184", Address: 388, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道消报"},
	{Field: "xhf0185", Address: 389, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道信号阀"},
	{Field: "slzs0186", Address: 390, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道水流指示"},
	{Field: "sg0187", Address: 391, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "1层走道声光"},
	{Field: "yg0188", Address: 392, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层办公室烟感"},
	{Field: "yg0189", Address: 393, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层办公室烟感"},
	{Field: "yg018a", Address: 394, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层测智室烟感"},
	{Field: "yg018b", Address: 395, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层测智室烟感"},
	{Field: "yg018c", Address: 396, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层儿科室烟感"},
	{Field: "yg018d", Address: 397, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层计划免疫烟感"},
	{Field: "yg018e", Address: 398, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层计划免疫烟感"},
	{Field: "yg018f", Address: 399, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层计划免疫烟感"},
	{Field: "yg0190", Address: 400, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层冷库烟感"},
	{Field: "yg0191", Address: 401, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层215房间烟感"},
	{Field: "yg0192", Address: 402, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层216房间烟感"},
	{Field: "yg0193", Address: 403, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层217房间烟感"},
	{Field: "yg0194", Address: 404, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层母婴室烟感"},
	{Field: "yg0195", Address: 405, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道烟感"},
	{Field: "yg0196", Address: 406, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道烟感"},
	{Field: "yg0197", Address: 407, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道烟感"},
	{Field: "yg0198", Address: 408, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道烟感"},
	{Field: "sb0199", Address: 409, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道手报"},
	{Field: "sb019a", Address: 410, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道手报"},
	{Field: "xb019b", Address: 411, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道消报"},
	{Field: "xb019c", Address: 412, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道消报"},
	{Field: "xhdf019d", Address: 413, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道信号蝶阀"},
	{Field: "slzs019e", Address: 414, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道水流指示"},
	{Field: "sg019f", Address: 415, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道声光"},
	{Field: "sg01a0", Address: 416, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "2层走道声光"},
	{Field: "yg0201", Address: 513, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层院长办公室烟感"},
	{Field: "yg0202", Address: 514, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层副院长办公室烟感"},
	{Field: "yg0203", Address: 515, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层示范性教育室烟感"},
	{Field: "yg0204", Address: 516, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层档案室烟感"},
	{Field: "yg0205", Address: 517, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层人事部烟感"},
	{Field: "yg0206", Address: 518, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层党支部烟感"},
	{Field: "yg0207", Address: 519, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层办公室烟感"},
	{Field: "yg0208", Address: 520, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层医疗办公室烟感"},
	{Field: "yg0209", Address: 521, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层财务室烟感"},
	{Field: "yg020a", Address: 522, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层楼梯口烟感"},
	{Field: "yg020b", Address: 523, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层电梯机房烟感"},
	{Field: "yg020c", Address: 524, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层仓库烟感"},
	{Field: "yg020d", Address: 525, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层会议室烟感"},
	{Field: "yg020e", Address: 526, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层会议室烟感"},
	{Field: "yg020f", Address: 527, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层楼梯口烟感"},
	{Field: "yg0210", Address: 528, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层医生值班室烟感"},
	{Field: "yg0211", Address: 529, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道烟感"},
	{Field: "yg0212", Address: 530, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道烟感"},
	{Field: "yg0213", Address: 531, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道烟感"},
	{Field: "yg0214", Address: 532, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道烟感"},
	{Field: "sb0215", Address: 533, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道手报"},
	{Field: "sg0216", Address: 534, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道声光"},
	{Field: "sb0217", Address: 535, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道手报"},
	{Field: "sg0218", Address: 536, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道声光"},
	{Field: "xb0219", Address: 537, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道消报"},
	{Field: "xb021a", Address: 538, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道消报"},
	{Field: "sl021b", Address: 539, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道水流"},
	{Field: "xhf021c", Address: 540, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "5层走道信号阀"},
	{Field: "yg021d", Address: 541, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层储物间烟感"},
	{Field: "yg021e", Address: 542, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg021f", Address: 543, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0220", Address: 544, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层楼梯口烟感"},
	{Field: "yg0221", Address: 545, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层vct室烟感"},
	{Field: "yg0222", Address: 546, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层信息管理烟感"},
	{Field: "yg0223", Address: 547, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层楼梯口烟感"},
	{Field: "yg0224", Address: 548, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0225", Address: 549, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0226", Address: 550, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0227", Address: 551, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg0228", Address: 552, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg0229", Address: 553, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg022a", Address: 554, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg022b", Address: 555, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg022c", Address: 556, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg022d", Address: 557, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg022e", Address: 558, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg022f", Address: 559, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg0230", Address: 560, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg0231", Address: 561, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层口腔烟感"},
	{Field: "yg0232", Address: 562, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层杂物间烟感"},
	{Field: "yg0233", Address: 563, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0234", Address: 564, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0235", Address: 565, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0236", Address: 566, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0237", Address: 567, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0238", Address: 568, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层诊室烟感"},
	{Field: "yg0239", Address: 569, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道烟感"},
	{Field: "yg023a", Address: 570, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道烟感"},
	{Field: "yg023b", Address: 571, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道烟感"},
	{Field: "yg023c", Address: 572, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道烟感"},
	{Field: "sb023d", Address: 573, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道手报"},
	{Field: "sb023e", Address: 574, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道手报"},
	{Field: "xb023f", Address: 575, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道消报"},
	{Field: "xb0240", Address: 576, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道消报"},
	{Field: "xb0241", Address: 577, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道消报"},
	{Field: "sl0242", Address: 578, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道水流"},
	{Field: "sl0243", Address: 579, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道水流"},
	{Field: "sg0244", Address: 580, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道声光"},
	{Field: "sg0245", Address: 581, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "4层走道声光"},
	{Field: "yg0246", Address: 582, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg0247", Address: 583, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg0248", Address: 584, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg0249", Address: 585, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg024a", Address: 586, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg024b", Address: 587, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg024c", Address: 588, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg024d", Address: 589, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg024e", Address: 590, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层治疗准备室烟感"},
	{Field: "yg024f", Address: 591, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层护士休息室烟感"},
	{Field: "yg0250", Address: 592, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层护士办公室烟感"},
	{Field: "yg0251", Address: 593, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层康复教室烟感"},
	{Field: "yg0252", Address: 594, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道烟感"},
	{Field: "yg0253", Address: 595, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层医生值班室烟感"},
	{Field: "yg0254", Address: 596, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层卫生间烟感"},
	{Field: "yg0255", Address: 597, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层医生办公室烟感"},
	{Field: "yg0256", Address: 598, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层医生办公室烟感"},
	{Field: "yg0257", Address: 599, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层抢救室烟感"},
	{Field: "yg0258", Address: 600, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg0259", Address: 601, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg025a", Address: 602, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层楼梯口烟感"},
	{Field: "yg025b", Address: 603, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg025c", Address: 604, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg025d", Address: 605, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg025e", Address: 606, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg025f", Address: 607, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg0260", Address: 608, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层病房烟感"},
	{Field: "yg0261", Address: 609, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层楼梯烟感"},
	{Field: "yg0262", Address: 610, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道烟感"},
	{Field: "yg0263", Address: 611, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层烟感"},
	{Field: "yg0264", Address: 612, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道烟感"},
	{Field: "yg0265", Address: 613, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道烟感"},
	{Field: "yg0266", Address: 614, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道烟感"},
	{Field: "yg0267", Address: 615, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层护士站烟感"},
	{Field: "sb0268", Address: 616, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道烟感"},
	{Field: "sb0269", Address: 617, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道手报"},
	{Field: "sb026a", Address: 618, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道手报"},
	{Field: "sb026b", Address: 619, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道手报"},
	{Field: "xb026c", Address: 620, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道消报"},
	{Field: "xb026d", Address: 621, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道消报"},
	{Field: "xb026e", Address: 622, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道消报"},
	{Field: "sl026f", Address: 623, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道水流"},
	{Field: "xhdf0270", Address: 624, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道信号蝶阀"},
	{Field: "sg0271", Address: 625, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道声光"},
	{Field: "sg0272", Address: 626, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道声光"},
	{Field: "sg0273", Address: 627, Scale: 1, Decimals: 0, RW: "R", Unit: "", Label: "3层走道声光"},
}

var addrToIndexes map[uint16][]int

func init() {
	addrToIndexes = make(map[uint16][]int)
	for i, p := range pointConfig {
		addrToIndexes[p.Address] = append(addrToIndexes[p.Address], i)
	}
}

//go:wasmexport handle
func handle() int32 {
	defer func() {
		if r := recover(); r != nil {
			outputJSON(map[string]interface{}{"success": false, "error": "panic"})
		}
	}()

	cfg := getConfig()
	points := readAllPoints(cfg.DeviceAddress, cfg.Debug)

	outputJSON(map[string]interface{}{
		"success": true,
		"points":  points,
	})
	return 0
}

//go:wasmexport describe
func describe() int32 {
	outputJSON(map[string]interface{}{
		"success": true,
		"data":    map[string]string{},
	})
	return 0
}

//go:wasmexport version
func version() int32 {
	outputJSON(map[string]interface{}{
		"success": true,
		"data": map[string]string{
			"version": DriverVersion,
		},
	})
	return 0
}

func readAllPoints(devAddr int, debug bool) []map[string]interface{} {
	points := make([]map[string]interface{}, 0, len(pointConfig))
	valueByAddr := make(map[uint16]uint16, len(pointConfig))

	ranges := []struct {
		Start uint16
		Count uint16
	}{
		{Start: 257, Count: 50},
		{Start: 307, Count: 50},
		{Start: 357, Count: 50},
		{Start: 407, Count: 10},
		{Start: 513, Count: 50},
		{Start: 563, Count: 50},
		{Start: 613, Count: 15},
	}

	for _, rg := range ranges {
		readRangeAdaptive(byte(devAddr), rg.Start, rg.Count, debug, valueByAddr)
	}

	if debug && len(valueByAddr) == 0 {
		logf("no points collected after adaptive reads")
	}

	for _, cfg := range pointConfig {
		raw, ok := valueByAddr[cfg.Address]
		if !ok {
			continue
		}
		realVal := float64(raw) * cfg.Scale
		points = append(points, map[string]interface{}{
			"field_name": cfg.Field,
			"value":      formatFloat(realVal, cfg.Decimals),
			"rw":         cfg.RW,
			"unit":       cfg.Unit,
			"label":      cfg.Label,
		})
	}
	return points
}

func readRangeAdaptive(devAddr byte, logicalStart uint16, count uint16, debug bool, out map[uint16]uint16) {
	if count == 0 {
		return
	}

	values := readMultipleRegsLogical(devAddr, logicalStart, count, debug)
	if values != nil {
		for i, v := range values {
			out[logicalStart+uint16(i)] = v
		}
		return
	}

	if count == 1 {
		if debug {
			logf("skip unreadable register=%d", logicalStart)
		}
		return
	}

	half := count / 2
	readRangeAdaptive(devAddr, logicalStart, half, debug, out)
	readRangeAdaptive(devAddr, logicalStart+half, count-half, debug, out)
}

func readMultipleRegsLogical(devAddr byte, logicalStart uint16, count uint16, debug bool) []uint16 {
	if values := readMultipleRegs(devAddr, logicalStart, count, debug); values != nil {
		return values
	}

	if logicalStart == 0 {
		return nil
	}

	if debug {
		logf("retry with 0-based address logical=%d query=%d count=%d", logicalStart, logicalStart-1, count)
	}

	return readMultipleRegs(devAddr, logicalStart-1, count, debug)
}

func readMultipleRegs(devAddr byte, startReg uint16, count uint16, debug bool) []uint16 {
	if count > 50 {
		count = 50
	}

	if values, err := readMultipleRegsWithFunc(devAddr, startReg, count, FUNC_CODE_READ_HOLDING, debug); err == nil {
		return values
	} else if debug {
		logf("read holding failed addr=%d count=%d err=%v", startReg, count, err)
	}

	if values, err := readMultipleRegsWithFunc(devAddr, startReg, count, FUNC_CODE_READ_INPUT, debug); err == nil {
		return values
	} else if debug {
		logf("read input failed addr=%d count=%d err=%v", startReg, count, err)
	}

	return nil
}

func readMultipleRegsWithFunc(devAddr byte, startReg uint16, count uint16, funcCode byte, debug bool) ([]uint16, error) {
	req := buildReadFrame(devAddr, startReg, count, funcCode)
	if debug {
		logf("rtu req fc=%02X % X", funcCode, req)
	}

	resp, n := serialTransceive(req, int(count)*2+5, 1000)
	if debug {
		logf("rtu fc=%02X n=%d resp=%s", funcCode, n, hexPreview(resp, n, 24))
	}
	if n <= 0 {
		return nil, errf("read timeout")
	}

	values, err := parseReadResponse(resp[:n], devAddr, funcCode)
	if err != nil {
		return nil, err
	}
	if len(values) < int(count) {
		return nil, errf("insufficient register data")
	}

	return values, nil
}

func serialTransceive(req []byte, respLen int, timeoutMs int) ([]byte, int) {
	if len(req) == 0 || respLen <= 0 {
		return nil, 0
	}
	reqMem := pdk.AllocateBytes(req)
	defer reqMem.Free()
	respMem := pdk.Allocate(respLen)
	defer respMem.Free()

	n := int(serial_transceive(
		reqMem.Offset(), uint64(len(req)),
		respMem.Offset(), uint64(respLen),
		uint64(timeoutMs),
	))
	if n <= 0 {
		return nil, n
	}
	if n > respLen {
		n = respLen
	}

	resp := make([]byte, n)
	mem := pdk.NewMemory(respMem.Offset(), uint64(n))
	mem.Load(resp)
	return resp, n
}

func buildReadFrame(addr byte, start uint16, qty uint16, funcCode byte) []byte {
	req := make([]byte, 8)
	req[0] = addr
	req[1] = funcCode
	req[2], req[3] = byte(start>>8), byte(start)
	req[4], req[5] = byte(qty>>8), byte(qty)
	crc := crc16(req[:6])
	req[6], req[7] = byte(crc), byte(crc>>8)
	return req
}

func parseReadResponse(data []byte, addr byte, funcCode byte) ([]uint16, error) {
	if len(data) < 5 || data[0] != addr {
		return nil, errf("invalid response")
	}
	if data[1] == (funcCode | 0x80) {
		return nil, errf("modbus exception code=" + strconv.Itoa(int(data[2])))
	}
	if data[1] != funcCode {
		return nil, errf("unexpected function code")
	}
	byteCnt := int(data[2])
	if byteCnt < 2 || len(data) < 3+byteCnt+2 {
		return nil, errf("byte count mismatch")
	}
	if !checkCRC(data[:3+byteCnt+2]) {
		return nil, errf("crc error")
	}

	values := make([]uint16, byteCnt/2)
	for i := 0; i < len(values); i++ {
		values[i] = uint16(data[3+i*2])<<8 | uint16(data[4+i*2])
	}
	return values, nil
}

func crc16(data []byte) uint16 {
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

func checkCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	got := uint16(data[len(data)-2]) | uint16(data[len(data)-1])<<8
	return crc16(data[:len(data)-2]) == got
}

func getConfig() DriverConfig {
	def := DriverConfig{DeviceAddress: 1, FuncName: "read"}
	var envelope struct {
		Config map[string]string `json:"config"`
	}
	if err := pdk.InputJSON(&envelope); err != nil {
		return def
	}

	cfg := def
	if v := strings.TrimSpace(envelope.Config["device_address"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DeviceAddress = n
		}
	}
	if v := strings.TrimSpace(envelope.Config["func_name"]); v != "" {
		cfg.FuncName = v
	}
	if v := strings.TrimSpace(envelope.Config["field_name"]); v != "" {
		cfg.FieldName = v
	}
	if v := strings.TrimSpace(envelope.Config["value"]); v != "" {
		cfg.Value = v
	}
	if v := strings.TrimSpace(envelope.Config["debug"]); v != "" {
		cfg.Debug = v == "1" || strings.EqualFold(v, "true")
	}
	return cfg
}

func formatFloat(val float64, decimals int) string {
	return strconv.FormatFloat(val, 'f', decimals, 64)
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }
func errf(s string) error         { return simpleErr(s) }

func outputJSON(v interface{}) {
	b, _ := json.Marshal(v)
	if len(b) == 0 {
		b = []byte(`{"success":false,"error":"encode failed"}`)
	}
	pdk.Output(b)
}

func logf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	pdk.Log(pdk.LogDebug, msg)
}

func hexPreview(b []byte, n int, max int) string {
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

func main() {}
