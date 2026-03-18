// Package hostio 负责把 TinyGo 驱动里的 Host I/O 模板收敛到一处。
//
// 这些驱动最终会被编译成 Extism/WASI 插件运行，真正的串口或 TCP 通信
// 不是在插件内直接完成，而是通过宿主注入的 wasmimport 函数完成。
// 每个驱动都需要做同样几件事：
// 1. 把请求字节拷贝到插件线性内存。
// 2. 为响应预留一块缓冲区。
// 3. 调用宿主函数。
// 4. 把宿主写回的响应拷回 Go 切片。
//
// 如果这些步骤散落在每个驱动文件里，会让真正重要的协议逻辑被样板代码淹没。
// 因此这里专门提供两个小函数，分别处理“返回新切片”和“写入现有缓冲区”两种场景。
package hostio

import pdk "github.com/extism/go-pdk"

type TransceiveFunc func(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

// TransceiveBytes 适用于“响应长度在调用前大致已知，但更希望拿到独立结果切片”的场景。
//
// 常见于 Modbus RTU：
// - 请求帧长度通常较小。
// - 响应长度会根据寄存器数量计算出来。
// - 调用方更关心“拿到完整响应字节后再解析”。
//
// 返回值中的 int 是宿主声称写入的实际字节数，保留这个数值有两个目的：
// 1. 便于调试日志输出真实长度。
// 2. 让上层在需要时能区分“空响应”“超时”“长度裁剪”等情况。
func TransceiveBytes(call TransceiveFunc, req []byte, respLen int, timeoutMs int) ([]byte, int) {
	if len(req) == 0 || respLen <= 0 {
		return nil, 0
	}

	reqMem := pdk.AllocateBytes(req)
	defer reqMem.Free()
	respMem := pdk.Allocate(respLen)
	defer respMem.Free()

	n := int(call(
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

// TransceiveInto 适用于“调用方自己持有响应缓冲区”的场景。
//
// 常见于 Modbus TCP：
// - 调用方通常会预先准备一块固定长度缓冲区。
// - 更关心在已有切片上原地填充，避免额外再分配一块结果切片。
//
// 它与 TransceiveBytes 的差别主要在于结果承载方式，底层内存搬运流程保持一致，
// 这样驱动层只需关心协议帧，不需要反复处理 Extism 的内存细节。
func TransceiveInto(call TransceiveFunc, req []byte, resp []byte, timeoutMs int) int {
	if len(req) == 0 || len(resp) == 0 {
		return 0
	}

	reqMem := pdk.AllocateBytes(req)
	defer reqMem.Free()
	respMem := pdk.Allocate(len(resp))
	defer respMem.Free()

	n := int(call(
		reqMem.Offset(), uint64(len(req)),
		respMem.Offset(), uint64(len(resp)),
		uint64(timeoutMs),
	))
	if n <= 0 {
		return n
	}
	if n > len(resp) {
		n = len(resp)
	}

	mem := pdk.NewMemory(respMem.Offset(), uint64(n))
	mem.Load(resp[:n])
	return n
}
