//! Rust Modbus RTU 驱动模板。
//!
//! 这个模板刻意保留了较多注释，目标不是代码最短，而是方便后续复制后直接改成真实驱动。
//! 如果你已经有一个 TinyGo 驱动，希望迁移到 Rust，通常只需要替换：
//! - 点表定义 POINT_CONFIGS
//! - 功能码常量 FUNC_CODE_READ
//! - 如果存在下行控制，再替换默认的可写点定义
//! - read_all_points 中的寄存器换算方式
//! - DRIVER_PRODUCT_KEY / DRIVER_VERSION
use extism_pdk::*;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

const DRIVER_VERSION: &str = "0.1.0";
const DRIVER_PRODUCT_KEY: &str = "rust_modbus_rtu_template";
const DRIVER_NAME: &str = "Rust Modbus RTU Driver Template";

const DEFAULT_DEVICE_ADDRESS: u8 = 1;
const DEFAULT_TIMEOUT_MS: u64 = 1000;
const FUNC_CODE_READ: u8 = 0x04;
const FUNC_CODE_WRITE_SINGLE: u8 = 0x06;

#[link(wasm_import_module = "extism:host/user")]
extern "C" {
    // 这个函数由宿主实现。
    // 插件负责：
    // 1. 准备请求帧
    // 2. 分配接收缓冲区
    // 3. 交给宿主完成一次串口收发
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
    // 当前网关真正稳定提供的是 config，因此模板只强依赖这一项。
    // 其他字段如 device_id、resource_type 如果未来有明确约定，再按需加回来即可。
    #[serde(default)]
    config: BTreeMap<String, String>,
}

#[derive(Debug, Serialize)]
struct DriverResponse {
    success: bool,
    #[serde(rename = "productKey")]
    product_key: String,
    points: Vec<DriverPoint>,
    #[serde(skip_serializing_if = "String::is_empty")]
    error: String,
}

#[derive(Debug, Serialize)]
struct DriverPoint {
    #[serde(rename = "field_name")]
    field_name: String,
    value: String,
    rw: String,
    unit: String,
    label: String,
}

#[derive(Debug, Serialize)]
struct DescribeResponse {
    success: bool,
    data: BTreeMap<String, String>,
}

#[derive(Debug, Serialize)]
struct VersionResponse {
    success: bool,
    data: BTreeMap<String, String>,
}

#[derive(Debug, Clone, Copy)]
struct PointConfig {
    // 一个 PointConfig 完整描述“一个业务点最终如何形成”。
    // Address/Length 说明去哪读；
    // Scale/Decimals 说明如何把原始寄存器值变成展示值；
    // RW/Unit/Label 则是输出给网关的业务元数据。
    field: &'static str,
    address: u16,
    length: u16,
    scale: f64,
    decimals: usize,
    rw: &'static str,
    unit: &'static str,
    label: &'static str,
}

// 模板默认点表故意做得很短，便于阅读：
// - 三个点刚好落在一段连续寄存器里
// - 都是单寄存器
// - 其中前两个点演示“可读可写”，第三个点保持只读
// - 都遵循“原值 * 0.1”的简单换算
// 这样迁移真实驱动时，可以同时对照读取和单点写入两条路径。
const POINT_CONFIGS: &[PointConfig] = &[
    PointConfig {
        field: "temperature",
        address: 0,
        length: 1,
        scale: 0.1,
        decimals: 1,
        rw: "RW",
        unit: "℃",
        label: "温度",
    },
    PointConfig {
        field: "humidity",
        address: 1,
        length: 1,
        scale: 0.1,
        decimals: 1,
        rw: "RW",
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
        label: "漏点温度",
    },
];

#[plugin_fn]
pub fn handle(input: String) -> FnResult<String> {
    // handle 是驱动主入口：
    // - 解析网关输入
    // - 执行读取
    // - 统一打包 success/productKey/points/error
    let request: DriverInvocationInput = serde_json::from_str(&input)
        .map_err(|err| Error::msg(format!("invalid input json: {err}")))?;

    let response = if is_write_func(&request.config) {
        match write_single_point(&request.config) {
            Ok(point) => DriverResponse {
                success: true,
                product_key: DRIVER_PRODUCT_KEY.to_string(),
                points: vec![point],
                error: String::new(),
            },
            Err(error) => DriverResponse {
                success: false,
                product_key: DRIVER_PRODUCT_KEY.to_string(),
                points: Vec::new(),
                error,
            },
        }
    } else {
        match read_all_points(&request.config) {
            Ok(points) => DriverResponse {
                success: true,
                product_key: DRIVER_PRODUCT_KEY.to_string(),
                points,
                error: String::new(),
            },
            Err(error) => DriverResponse {
                success: false,
                product_key: DRIVER_PRODUCT_KEY.to_string(),
                points: Vec::new(),
                error,
            },
        }
    };

    Ok(serde_json::to_string(&response)
        .map_err(|err| Error::msg(format!("serialize handle output failed: {err}")))?)
}

#[plugin_fn]
pub fn describe() -> FnResult<String> {
    // describe 返回的是“如何使用这个驱动”的静态说明。
    // 这里不追求复杂结构，而是用最稳定的字符串 map，方便宿主兼容。
    let mut data = BTreeMap::new();
    data.insert("name".into(), DRIVER_NAME.into());
    data.insert("language".into(), "rust".into());
    data.insert("transport".into(), "serial_transceive".into());
    data.insert("template".into(), "modbus_rtu_single_write".into());
    data.insert("device_address".into(), "Modbus 从站地址，默认 1".into());
    data.insert("timeout_ms".into(), "串口超时毫秒，默认 1000".into());
    data.insert("debug".into(), "true/false，可选".into());
    data.insert("func_name".into(), "read 或 write".into());
    data.insert("field_name".into(), "write 模式下必填，只能一次写一个字段".into());
    data.insert("value".into(), "write 模式下必填，传工程量字符串".into());
    data.insert("writable_fields".into(), writable_fields_text());

    Ok(serde_json::to_string(&DescribeResponse {
        success: true,
        data,
    })
    .map_err(|err| Error::msg(format!("serialize describe output failed: {err}")))?)
}

#[plugin_fn]
pub fn version() -> FnResult<String> {
    // version 只保留最稳定的最小字段，用于网关识别驱动版本。
    let mut data = BTreeMap::new();
    data.insert("version".into(), DRIVER_VERSION.into());
    data.insert("productKey".into(), DRIVER_PRODUCT_KEY.into());

    Ok(serde_json::to_string(&VersionResponse {
        success: true,
        data,
    })
    .map_err(|err| Error::msg(format!("serialize version output failed: {err}")))?)
}

fn read_all_points(config: &BTreeMap<String, String>) -> Result<Vec<DriverPoint>, String> {
    if POINT_CONFIGS.is_empty() {
        return Err("point config is empty".into());
    }

    let device_address = parse_u8_config(config.get("device_address"), DEFAULT_DEVICE_ADDRESS);
    let timeout_ms = parse_u64_config(config.get("timeout_ms"), DEFAULT_TIMEOUT_MS);
    let debug = parse_bool_config(config.get("debug"));

    // 这里选择“扫描点表 -> 自动计算连续读取区间”，原因是：
    // - 点表仍然是唯一事实来源
    // - 新增点位时不需要再手工同步一份总长度
    // - 对于连续寄存器型设备，可读性通常优于散落的硬编码常量
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

    let request_frame = build_read_frame(device_address, start_address, quantity);
    if debug {
        debug!("rtu req={}", bytes_to_hex(&request_frame));
    }

    let response_capacity = expected_response_len(quantity);
    let response = serial_roundtrip(&request_frame, response_capacity, timeout_ms)?;
    if debug {
        debug!("rtu resp={}", bytes_to_hex(&response));
    }

    let registers = parse_read_response(&response, device_address, FUNC_CODE_READ)?;

    // 通信层拿到的是一个连续寄存器切片；
    // 业务层真正关心的，是如何按点表把这些寄存器恢复成统一 points。
    let mut points = Vec::with_capacity(POINT_CONFIGS.len());
    for point in POINT_CONFIGS {
        let offset = usize::from(point.address - start_address);
        if offset >= registers.len() {
            continue;
        }

        // 模板默认只处理单寄存器点位。
        // 如果你的设备存在 2 寄存器或 4 寄存器的整型/浮点数，
        // 建议在这里单独抽一个 decode_u32 / decode_i32 / decode_f32 帮助函数。
        let raw = registers[offset];
        let value = f64::from(raw) * point.scale;

        points.push(DriverPoint {
            field_name: point.field.to_string(),
            value: format_scaled_value(value, point.decimals),
            rw: point.rw.to_string(),
            unit: point.unit.to_string(),
            label: point.label.to_string(),
        });
    }

    Ok(points)
}

fn write_single_point(config: &BTreeMap<String, String>) -> Result<DriverPoint, String> {
    let device_address = parse_u8_config(config.get("device_address"), DEFAULT_DEVICE_ADDRESS);
    let timeout_ms = parse_u64_config(config.get("timeout_ms"), DEFAULT_TIMEOUT_MS);
    let debug = parse_bool_config(config.get("debug"));

    let field_name = config
        .get("field_name")
        .map(|value| value.trim())
        .filter(|value| !value.is_empty())
        .ok_or_else(|| "write config missing field_name".to_string())?;
    let value_text = config
        .get("value")
        .map(|value| value.trim())
        .filter(|value| !value.is_empty())
        .ok_or_else(|| "write config missing value".to_string())?;

    let point = find_writable_point(field_name)?;
    if point.length != 1 {
        return Err(format!(
            "writable field {} requires {} registers, template only supports single-register writes",
            point.field, point.length
        ));
    }

    let raw_value = encode_write_value(point, value_text)?;
    let request_frame = build_write_single_frame(device_address, point.address, raw_value);
    if debug {
        debug!("rtu write req={}", bytes_to_hex(&request_frame));
    }

    let response = serial_roundtrip(&request_frame, 8, timeout_ms)?;
    if debug {
        debug!("rtu write resp={}", bytes_to_hex(&response));
    }

    let (written_register, written_value) =
        parse_write_single_response(&response, device_address, FUNC_CODE_WRITE_SINGLE)?;
    if written_register != point.address {
        return Err(format!(
            "write register mismatch: expected {}, got {}",
            point.address, written_register
        ));
    }
    if written_value != raw_value {
        return Err(format!(
            "write value mismatch: expected {}, got {}",
            raw_value, written_value
        ));
    }

    Ok(DriverPoint {
        field_name: point.field.to_string(),
        value: format_scaled_value(f64::from(written_value) * point.scale, point.decimals),
        rw: point.rw.to_string(),
        unit: point.unit.to_string(),
        label: point.label.to_string(),
    })
}

fn build_read_frame(device_address: u8, start_address: u16, quantity: u16) -> Vec<u8> {
    // 组装标准 Modbus RTU 读寄存器请求：
    // [设备地址][功能码][起始地址高][起始地址低][数量高][数量低][CRC低][CRC高]
    let mut frame = vec![
        device_address,
        FUNC_CODE_READ,
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

fn build_write_single_frame(device_address: u8, register: u16, value: u16) -> Vec<u8> {
    // 标准 0x06 单寄存器写帧：
    // [设备地址][功能码][寄存器高][寄存器低][值高][值低][CRC低][CRC高]
    let mut frame = vec![
        device_address,
        FUNC_CODE_WRITE_SINGLE,
        (register >> 8) as u8,
        register as u8,
        (value >> 8) as u8,
        value as u8,
    ];
    let crc = crc16(&frame);
    frame.push((crc & 0x00ff) as u8);
    frame.push((crc >> 8) as u8);
    frame
}

fn parse_read_response(
    response: &[u8],
    device_address: u8,
    function_code: u8,
) -> Result<Vec<u16>, String> {
    // 这里优先做最值得排障的几类校验：
    // - 帧太短
    // - 地址不匹配
    // - 功能码不匹配
    // - 长度不匹配
    // - CRC 不通过
    if response.len() < 5 {
        return Err("response too short".into());
    }
    if response[0] != device_address {
        return Err("device address mismatch".into());
    }
    if response[1] != function_code {
        return Err("function code mismatch".into());
    }

    let byte_count = usize::from(response[2]);
    if byte_count % 2 != 0 {
        return Err("byte count must be even".into());
    }

    let expected_len = 3 + byte_count + 2;
    if response.len() < expected_len {
        return Err("response length mismatch".into());
    }
    if !check_crc(&response[..expected_len]) {
        return Err("crc check failed".into());
    }

    let payload = &response[3..3 + byte_count];
    let mut registers = Vec::with_capacity(byte_count / 2);
    let mut index = 0;
    while index < payload.len() {
        let value = u16::from(payload[index]) << 8 | u16::from(payload[index + 1]);
        registers.push(value);
        index += 2;
    }
    Ok(registers)
}

fn parse_write_single_response(
    response: &[u8],
    device_address: u8,
    function_code: u8,
) -> Result<(u16, u16), String> {
    if response.len() < 8 {
        return Err("write response too short".into());
    }
    if response[0] != device_address {
        return Err("write response device address mismatch".into());
    }
    if response[1] != function_code {
        return Err("write response function code mismatch".into());
    }
    if !check_crc(&response[..8]) {
        return Err("write response crc check failed".into());
    }

    let register = u16::from(response[2]) << 8 | u16::from(response[3]);
    let value = u16::from(response[4]) << 8 | u16::from(response[5]);
    Ok((register, value))
}

fn serial_roundtrip(
    request: &[u8],
    response_capacity: usize,
    timeout_ms: u64,
) -> Result<Vec<u8>, String> {
    if request.is_empty() {
        return Err("empty request frame".into());
    }
    if response_capacity == 0 {
        return Err("response capacity must be > 0".into());
    }

    let mut response = vec![0u8; response_capacity];
    let written = unsafe {
        serial_transceive(
            request.as_ptr() as u64,
            request.len() as u64,
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
    // RTU 正常读响应长度：
    // 设备地址 1 + 功能码 1 + 字节数 1 + 数据区 quantity*2 + CRC 2
    3 + usize::from(quantity) * 2 + 2
}

fn parse_u8_config(value: Option<&String>, default: u8) -> u8 {
    value
        .and_then(|text| text.trim().parse::<u8>().ok())
        .unwrap_or(default)
}

fn parse_u64_config(value: Option<&String>, default: u64) -> u64 {
    value
        .and_then(|text| text.trim().parse::<u64>().ok())
        .filter(|v| *v > 0)
        .unwrap_or(default)
}

fn parse_bool_config(value: Option<&String>) -> bool {
    value
        .map(|text| {
            let normalized = text.trim();
            normalized.eq_ignore_ascii_case("true")
                || normalized.eq_ignore_ascii_case("yes")
                || normalized == "1"
        })
        .unwrap_or(false)
}

fn parse_f64_config(value: &str) -> Result<f64, String> {
    value
        .trim()
        .parse::<f64>()
        .map_err(|err| format!("invalid write value: {err}"))
}

fn is_write_func(config: &BTreeMap<String, String>) -> bool {
    config
        .get("func_name")
        .map(|value| value.trim().eq_ignore_ascii_case("write"))
        .unwrap_or(false)
}

fn writable_fields_text() -> String {
    POINT_CONFIGS
        .iter()
        .filter(|point| point.rw.to_ascii_uppercase().contains('W'))
        .map(|point| point.field)
        .collect::<Vec<_>>()
        .join(",")
}

fn find_writable_point(field_name: &str) -> Result<&'static PointConfig, String> {
    let normalized = field_name.trim();
    let point = POINT_CONFIGS
        .iter()
        .find(|point| point.field.eq_ignore_ascii_case(normalized))
        .ok_or_else(|| format!("unsupported writable field: {normalized}"))?;

    if !point.rw.to_ascii_uppercase().contains('W') {
        return Err(format!("field {} is not writable", point.field));
    }

    Ok(point)
}

fn encode_write_value(point: &PointConfig, value_text: &str) -> Result<u16, String> {
    if point.scale == 0.0 {
        return Err(format!("field {} has invalid scale 0", point.field));
    }

    let engineering_value = parse_f64_config(value_text)?;
    let raw_value = (engineering_value / point.scale).round();
    if !(0.0..=65535.0).contains(&raw_value) {
        return Err(format!(
            "write value out of range for field {}: {}",
            point.field, engineering_value
        ));
    }
    Ok(raw_value as u16)
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
    // 输出风格尽量贴近当前 TinyGo 驱动：
    // - 保留需要的小数位
    // - 自动去掉末尾无意义的 0
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

        let registers = parse_read_response(&response, 0x01, 0x04).unwrap();
        assert_eq!(registers, vec![0x00FD, 0x01F9]);
    }

    #[test]
    fn build_write_single_frame_appends_crc() {
        let frame = build_write_single_frame(0x01, 0x0010, 0x1234);
        assert_eq!(frame, vec![0x01, 0x06, 0x00, 0x10, 0x12, 0x34, 0x85, 0x78]);
    }

    #[test]
    fn parse_write_single_response_returns_echo() {
        let mut response = vec![0x01, 0x06, 0x00, 0x10, 0x12, 0x34];
        let crc = crc16(&response);
        response.push((crc & 0x00ff) as u8);
        response.push((crc >> 8) as u8);

        let (register, value) = parse_write_single_response(&response, 0x01, 0x06).unwrap();
        assert_eq!(register, 0x0010);
        assert_eq!(value, 0x1234);
    }

    #[test]
    fn encode_write_value_rounds_engineering_value() {
        let point = find_writable_point("temperature").unwrap();
        assert_eq!(encode_write_value(point, "23.6").unwrap(), 236);
    }

    #[test]
    fn find_writable_point_rejects_readonly_field() {
        let error = find_writable_point("dewtemperature").unwrap_err();
        assert!(error.contains("not writable"));
    }

    #[test]
    fn format_scaled_value_trims_trailing_zeros() {
        assert_eq!(format_scaled_value(12.3000, 3), "12.3");
        assert_eq!(format_scaled_value(12.0, 1), "12");
        assert_eq!(format_scaled_value(12.0, 0), "12");
    }

    #[test]
    fn parse_bool_config_supports_common_true_values() {
        assert!(parse_bool_config(Some(&"true".to_string())));
        assert!(parse_bool_config(Some(&"1".to_string())));
        assert!(parse_bool_config(Some(&"yes".to_string())));
        assert!(!parse_bool_config(Some(&"false".to_string())));
    }
}
