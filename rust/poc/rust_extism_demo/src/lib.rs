//! 一个偏“教学用途”的 Rust Extism 驱动模板。
//!
//! 这个 POC 的目标不是追求最少代码，而是把一个只读 Modbus RTU 驱动所需的关键环节
//! 都完整铺开，方便后续把点表和协议替换成真实设备：
//! - 解析网关输入
//! - 构造 RTU 请求帧
//! - 通过宿主提供的 serial_transceive 做一次串口往返
//! - 解析寄存器响应
//! - 按点表配置生成统一 points 输出
use extism_pdk::*;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

const DRIVER_VERSION: &str = "0.2.0";
const DRIVER_PRODUCT_KEY: &str = "rust_modbus_rtu_template";
const DRIVER_NAME: &str = "Rust Modbus RTU Template";
const DEFAULT_TIMEOUT_MS: u64 = 1000;
const FUNC_CODE_READ_INPUT: u8 = 0x04;

#[link(wasm_import_module = "extism:host/user")]
extern "C" {
    // 这个函数由宿主提供，插件本身只负责准备请求和接收响应。
    fn serial_transceive(
        write_ptr: u64,
        write_size: u64,
        read_ptr: u64,
        read_cap: u64,
        timeout_ms: u64,
    ) -> u64;
}

#[derive(Debug, Deserialize)]
struct DriverInvocationInput {
    // 模板里保留这些字段，主要是为了说明真实驱动在运行时能拿到哪些上下文。
    // 具体业务是否使用，取决于设备和网关的对接方式。
    #[serde(default)]
    device_id: i64,
    #[serde(default)]
    device_name: String,
    #[serde(default)]
    resource_id: i64,
    #[serde(default)]
    resource_type: String,
    #[serde(default)]
    config: BTreeMap<String, String>,
    #[serde(default)]
    device_config: String,
}

#[derive(Debug, Serialize)]
struct DriverResponse {
    success: bool,
    #[serde(rename = "productKey")]
    product_key: String,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    points: Vec<DriverPoint>,
    #[serde(skip_serializing_if = "String::is_empty")]
    error: String,
}

#[derive(Debug, Serialize)]
struct DriverPoint {
    field_name: String,
    value: String,
    rw: String,
    unit: String,
    label: String,
}

#[derive(Debug, Serialize)]
struct VersionResponse {
    success: bool,
    data: BTreeMap<String, String>,
}

#[derive(Debug, Serialize)]
struct DescribeResponse {
    success: bool,
    data: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Copy)]
struct PointConfig {
    // PointConfig 把“寄存器定义”和“点位元数据”绑在一起，方便直接按表生成输出。
    field: &'static str,
    address: u16,
    length: u16,
    scale: f64,
    decimals: usize,
    rw: &'static str,
    unit: &'static str,
    label: &'static str,
}

// 模板点表：真实项目通常只需要替换这一块和下方的功能码/缩放逻辑。
const POINT_CONFIGS: &[PointConfig] = &[
    PointConfig {
        field: "temperature",
        address: 0,
        length: 1,
        scale: 0.1,
        decimals: 1,
        rw: "R",
        unit: "℃",
        label: "温度",
    },
    PointConfig {
        field: "humidity",
        address: 1,
        length: 1,
        scale: 0.1,
        decimals: 1,
        rw: "R",
        unit: "%",
        label: "湿度",
    },
    PointConfig {
        field: "dewtemperature",
        address: 2,
        length: 1,
        scale: 0.1,
        decimals: 1,
        rw: "R",
        unit: "℃",
        label: "露点温度",
    },
];

#[plugin_fn]
pub fn handle(input: String) -> FnResult<String> {
    // handle 是驱动的主入口：解析输入，读取设备，最后输出统一 JSON。
    let request: DriverInvocationInput = serde_json::from_str(&input)
        .map_err(|e| Error::msg(format!("invalid input json: {e}")))?;

    let response = match try_handle(&request) {
        Ok(points) => DriverResponse {
            success: true,
            product_key: DRIVER_PRODUCT_KEY.to_string(),
            points,
            error: String::new(),
        },
        Err(err) => DriverResponse {
            success: false,
            product_key: DRIVER_PRODUCT_KEY.to_string(),
            points: Vec::new(),
            error: err,
        },
    };

    Ok(
        serde_json::to_string(&response)
            .map_err(|e| Error::msg(format!("serialize output failed: {e}")))?,
    )
}

#[plugin_fn]
pub fn describe() -> FnResult<String> {
    // describe 主要告诉调用方这个模板需要哪些 config，以及它适合什么传输方式。
    let mut data = BTreeMap::new();
    data.insert("name".into(), DRIVER_NAME.into());
    data.insert("language".into(), "rust".into());
    data.insert("transport".into(), "serial_transceive".into());
    data.insert("template".into(), "modbus_rtu_readonly".into());
    data.insert("device_address".into(), "Modbus 从站地址，必填".into());
    data.insert("timeout_ms".into(), "串口超时毫秒，默认 1000".into());
    data.insert("debug".into(), "true/false，可选".into());
    data.insert("notes".into(), "修改 POINT_CONFIGS、FUNC_CODE_READ_INPUT 和缩放逻辑即可复用".into());

    Ok(
        serde_json::to_string(&DescribeResponse { success: true, data })
            .map_err(|e| Error::msg(format!("serialize describe failed: {e}")))?,
    )
}

#[plugin_fn]
pub fn version() -> FnResult<String> {
    // version 返回一个非常稳定的小结构，供网关做驱动识别和版本展示。
    let mut data = BTreeMap::new();
    data.insert("version".into(), DRIVER_VERSION.into());
    data.insert("productKey".into(), DRIVER_PRODUCT_KEY.into());

    Ok(
        serde_json::to_string(&VersionResponse { success: true, data })
            .map_err(|e| Error::msg(format!("serialize version failed: {e}")))?,
    )
}

fn try_handle(request: &DriverInvocationInput) -> Result<Vec<DriverPoint>, String> {
    // 这里先做与设备无关的基本校验，再进入真正的寄存器读取流程。
    if request.resource_type.trim() != "serial" {
        return Err("modbus rtu 模板仅支持 serial 资源".into());
    }
    if POINT_CONFIGS.is_empty() {
        return Err("point config is empty".into());
    }

    let device_address = parse_required_u8(&request.config, "device_address")?;
    let timeout_ms = parse_u64_config(request.config.get("timeout_ms"), DEFAULT_TIMEOUT_MS);
    let debug = parse_bool_config(request.config.get("debug"));

    if debug {
        debug!(
            "handle device_id={} device_name={} resource_id={} device_config_len={}",
            request.device_id,
            request.device_name.trim(),
            request.resource_id,
            request.device_config.len()
        );
    }

    read_all_points(device_address, timeout_ms, debug)
}

fn read_all_points(device_address: u8, timeout_ms: u64, debug: bool) -> Result<Vec<DriverPoint>, String> {
    // 通过扫描点表自动计算连续读取区间，避免手工维护“起始地址 + 总长度”两份信息。
    let start_address = POINT_CONFIGS
        .iter()
        .map(|point| point.address)
        .min()
        .ok_or_else(|| "point config is empty".to_string())?;
    let end_address = POINT_CONFIGS
        .iter()
        .map(|point| point.address + point.length)
        .max()
        .ok_or_else(|| "point config is empty".to_string())?;
    let quantity = end_address - start_address;

    let request = build_read_frame(device_address, start_address, quantity);
    if debug {
        debug!("rtu req={}", bytes_to_hex(&request));
    }

    let response_capacity = expected_response_len(quantity);
    let response = serial_roundtrip(&request, response_capacity, timeout_ms)?;
    if debug {
        debug!("rtu resp={}", bytes_to_hex(&response));
    }

    let registers = parse_read_response(&response, device_address, FUNC_CODE_READ_INPUT)?;
    // 拿到连续寄存器后，再按点表把每个位置翻译成业务输出。
    let mut points = Vec::with_capacity(POINT_CONFIGS.len());
    for point in POINT_CONFIGS {
        let offset = usize::from(point.address - start_address);
        if offset >= registers.len() {
            continue;
        }

        let raw = registers[offset];
        let scaled = f64::from(raw) * point.scale;
        points.push(DriverPoint {
            field_name: point.field.to_string(),
            value: format_scaled_value(scaled, point.decimals),
            rw: point.rw.to_string(),
            unit: point.unit.to_string(),
            label: point.label.to_string(),
        });
    }

    Ok(points)
}

fn build_read_frame(device_address: u8, start_address: u16, quantity: u16) -> Vec<u8> {
    // RTU 帧结构固定，因此直接按协议顺序依次写入即可。
    let mut frame = vec![
        device_address,
        FUNC_CODE_READ_INPUT,
        (start_address >> 8) as u8,
        start_address as u8,
        (quantity >> 8) as u8,
        quantity as u8,
    ];
    let crc = crc16(&frame);
    frame.push((crc & 0x00ff) as u8);
    frame.push((crc >> 8) as u8);
    frame
}

fn parse_read_response(data: &[u8], device_address: u8, function_code: u8) -> Result<Vec<u16>, String> {
    // 读取响应时优先区分“太短”“功能码异常”“CRC 错误”等情况，
    // 这样后续排查串口参数或设备地址错误时更容易定位。
    if data.len() < 5 {
        return Err("response too short".into());
    }
    if data[0] != device_address {
        return Err("device address mismatch".into());
    }
    if data[1] != function_code {
        return Err("function code mismatch".into());
    }

    let byte_count = data[2] as usize;
    let expected_len = 3 + byte_count + 2;
    if data.len() < expected_len {
        return Err("response length mismatch".into());
    }
    if !check_crc(&data[..expected_len]) {
        return Err("crc check failed".into());
    }
    if byte_count % 2 != 0 {
        return Err("byte count must be even".into());
    }

    let mut registers = Vec::with_capacity(byte_count / 2);
    let payload = &data[3..3 + byte_count];
    let mut index = 0;
    while index < payload.len() {
        let value = u16::from(payload[index]) << 8 | u16::from(payload[index + 1]);
        registers.push(value);
        index += 2;
    }
    Ok(registers)
}

fn serial_roundtrip(frame: &[u8], response_len: usize, timeout_ms: u64) -> Result<Vec<u8>, String> {
    if frame.is_empty() {
        return Err("empty request frame".into());
    }
    if response_len == 0 {
        return Err("response_len must be > 0".into());
    }

    let mut response = vec![0u8; response_len];
    let written = unsafe {
        serial_transceive(
            frame.as_ptr() as u64,
            frame.len() as u64,
            response.as_mut_ptr() as u64,
            response.len() as u64,
            timeout_ms,
        ) as usize
    };

    if written == 0 {
        return Err("serial_transceive returned empty response".into());
    }
    if written > response.len() {
        return Err("serial_transceive returned invalid response length".into());
    }

    response.truncate(written);
    Ok(response)
}

fn expected_response_len(quantity: u16) -> usize {
    3 + usize::from(quantity) * 2 + 2
}

fn parse_required_u8(config: &BTreeMap<String, String>, key: &str) -> Result<u8, String> {
    config
        .get(key)
        .map(|value| value.trim())
        .filter(|value| !value.is_empty())
        .ok_or_else(|| format!("missing required config: {key}"))?
        .parse::<u8>()
        .map_err(|_| format!("invalid u8 config: {key}"))
}

fn parse_u64_config(value: Option<&String>, default: u64) -> u64 {
    value
        .and_then(|s| s.trim().parse::<u64>().ok())
        .filter(|v| *v > 0)
        .unwrap_or(default)
}

fn parse_bool_config(value: Option<&String>) -> bool {
    value
        .map(|s| {
            let normalized = s.trim();
            normalized.eq_ignore_ascii_case("true") || normalized == "1" || normalized.eq_ignore_ascii_case("yes")
        })
        .unwrap_or(false)
}

fn crc16(data: &[u8]) -> u16 {
    let mut crc: u16 = 0xffff;
    for byte in data {
        crc ^= u16::from(*byte);
        for _ in 0..8 {
            if crc & 0x0001 != 0 {
                crc = (crc >> 1) ^ 0xa001;
            } else {
                crc >>= 1;
            }
        }
    }
    crc
}

fn check_crc(data: &[u8]) -> bool {
    if data.len() < 2 {
        return false;
    }
    let expected = crc16(&data[..data.len() - 2]);
    let actual = u16::from(data[data.len() - 2]) | (u16::from(data[data.len() - 1]) << 8);
    expected == actual
}

fn format_scaled_value(value: f64, decimals: usize) -> String {
    let mut rendered = format!("{value:.decimals$}");
    if decimals == 0 {
        return rendered;
    }
    while rendered.contains('.') && rendered.ends_with('0') {
        rendered.pop();
    }
    if rendered.ends_with('.') {
        rendered.pop();
    }
    rendered
}

fn bytes_to_hex(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        out.push(hex_char(byte >> 4));
        out.push(hex_char(byte & 0x0f));
    }
    out
}

fn hex_char(nibble: u8) -> char {
    match nibble {
        0..=9 => (b'0' + nibble) as char,
        _ => (b'A' + (nibble - 10)) as char,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn build_read_frame_appends_crc() {
        let frame = build_read_frame(0x01, 0x0000, 0x0002);
        assert_eq!(frame, vec![0x01, 0x04, 0x00, 0x00, 0x00, 0x02, 0x71, 0xCB]);
    }

    #[test]
    fn parse_read_response_returns_registers() {
        let mut response = vec![0x01, 0x04, 0x04, 0x00, 0xFD, 0x01, 0xF9];
        let crc = crc16(&response);
        response.push((crc & 0x00ff) as u8);
        response.push((crc >> 8) as u8);
        let values = parse_read_response(&response, 0x01, 0x04).unwrap();
        assert_eq!(values, vec![0x00FD, 0x01F9]);
    }

    #[test]
    fn format_scaled_value_trims_trailing_zeros() {
        assert_eq!(format_scaled_value(12.3000, 3), "12.3");
        assert_eq!(format_scaled_value(12.0, 1), "12");
        assert_eq!(format_scaled_value(12.0, 0), "12");
    }
}
