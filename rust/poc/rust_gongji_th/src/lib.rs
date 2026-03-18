//! Rust 版的共济温湿度驱动示例。
//!
//! 相比 rust_extism_demo，这个文件更接近一个“真实设备驱动”的最小实现：
//! - 输入只保留真正使用到的 config
//! - 点表固定且简单
//! - 读流程直接围绕 3 个寄存器展开
//! 它适合作为 Rust 驱动风格和输出约定的基线参考。
use extism_pdk::*;
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

const DRIVER_VERSION: &str = "1.0.0-rust";
const DRIVER_PRODUCT_KEY: &str = "ljzchc_gongji_th";
const DEFAULT_DEVICE_ADDRESS: u8 = 1;
const DEFAULT_TIMEOUT_MS: u64 = 1000;
const FUNC_CODE_READ_INPUT: u8 = 0x04;

#[link(wasm_import_module = "extism:host/user")]
extern "C" {
    // 由宿主实现的串口往返调用。插件本身不直接持有串口。
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
    #[serde(default)]
    config: BTreeMap<String, String>,
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
    // 每个点只关心地址、缩放方式和输出展示信息。
    field: &'static str,
    address: u16,
    scale: f64,
    decimals: usize,
    rw: &'static str,
    unit: &'static str,
    label: &'static str,
}

// 共济温湿度设备的点表很短，因此直接写成常量数组最直观。
const POINTS: &[PointConfig] = &[
    PointConfig {
        field: "temperature",
        address: 0,
        scale: 0.1,
        decimals: 1,
        rw: "R",
        unit: "℃",
        label: "温度",
    },
    PointConfig {
        field: "humidity",
        address: 1,
        scale: 0.1,
        decimals: 1,
        rw: "R",
        unit: "%",
        label: "湿度",
    },
    PointConfig {
        field: "dewtemperature",
        address: 2,
        scale: 0.1,
        decimals: 1,
        rw: "R",
        unit: "℃",
        label: "漏点温度",
    },
];

#[plugin_fn]
pub fn handle(input: String) -> FnResult<String> {
    // 入口函数的职责非常单一：反序列化输入，读取点位，回包。
    let request: DriverInvocationInput = serde_json::from_str(&input)
        .map_err(|e| Error::msg(format!("invalid input json: {e}")))?;

    let response = match read_points(&request.config) {
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
    // 当前驱动没有可写字段，因此 describe 只返回一个空 data。
    let data = BTreeMap::new();
    Ok(
        serde_json::to_string(&DescribeResponse { success: true, data })
            .map_err(|e| Error::msg(format!("serialize describe failed: {e}")))?,
    )
}

#[plugin_fn]
pub fn version() -> FnResult<String> {
    // version 供网关识别驱动版本和 productKey。
    let mut data = BTreeMap::new();
    data.insert("version".into(), DRIVER_VERSION.into());
    data.insert("productKey".into(), DRIVER_PRODUCT_KEY.into());

    Ok(
        serde_json::to_string(&VersionResponse { success: true, data })
            .map_err(|e| Error::msg(format!("serialize version failed: {e}")))?,
    )
}

fn read_points(config: &BTreeMap<String, String>) -> Result<Vec<DriverPoint>, String> {
    // 这个设备寄存器非常规整，因此直接固定读取 0~2 共 3 个寄存器。
    let device_address = parse_u8_config(config.get("device_address"), DEFAULT_DEVICE_ADDRESS);
    let timeout_ms = parse_u64_config(config.get("timeout_ms"), DEFAULT_TIMEOUT_MS);
    let debug = parse_bool_config(config.get("debug"));

    let request = build_read_frame(device_address, 0, 3);
    if debug {
        debug!("rtu req={}", hex_preview(&request));
    }

    let response = serial_roundtrip(&request, 11, timeout_ms)?;
    if debug {
        debug!("rtu resp={}", hex_preview(&response));
    }

    let values = parse_read_response(&response, device_address)?;
    if values.len() < 3 {
        return Ok(Vec::new());
    }

    // 再按点表顺序把寄存器翻译成最终业务点。
    let mut points = Vec::with_capacity(POINTS.len());
    for point in POINTS {
        let raw = values[usize::from(point.address)];
        let real = f64::from(raw) * point.scale;
        points.push(DriverPoint {
            field_name: point.field.to_string(),
            value: format_scaled_value(real, point.decimals),
            rw: point.rw.to_string(),
            unit: point.unit.to_string(),
            label: point.label.to_string(),
        });
    }

    Ok(points)
}

fn serial_roundtrip(request: &[u8], response_len: usize, timeout_ms: u64) -> Result<Vec<u8>, String> {
    // 这里把一次宿主调用收敛成单独函数，目的是让主流程更像“协议代码”而不是“宿主适配代码”。
    let mut response = vec![0u8; response_len];
    let n = unsafe {
        serial_transceive(
            request.as_ptr() as u64,
            request.len() as u64,
            response.as_mut_ptr() as u64,
            response.len() as u64,
            timeout_ms,
        ) as usize
    };

    if n == 0 {
        return Err("serial_transceive returned empty response".into());
    }
    if n > response.len() {
        return Err("serial_transceive returned invalid response length".into());
    }
    response.truncate(n);
    Ok(response)
}

fn build_read_frame(device_address: u8, start: u16, count: u16) -> Vec<u8> {
    // 组装标准 RTU 读输入寄存器请求帧。
    let mut frame = vec![
        device_address,
        FUNC_CODE_READ_INPUT,
        (start >> 8) as u8,
        start as u8,
        (count >> 8) as u8,
        count as u8,
    ];
    let crc = crc16(&frame);
    frame.push((crc & 0x00ff) as u8);
    frame.push((crc >> 8) as u8);
    frame
}

fn parse_read_response(response: &[u8], device_address: u8) -> Result<Vec<u16>, String> {
    // 这里保留最必要的协议校验：地址、功能码、字节数和 CRC。
    if response.len() < 5 {
        return Err("invalid response".into());
    }
    if response[0] != device_address || response[1] != FUNC_CODE_READ_INPUT {
        return Err("invalid response".into());
    }

    let byte_count = response[2] as usize;
    if byte_count < 2 || response.len() < 3 + byte_count + 2 {
        return Err("byte count mismatch".into());
    }
    if !check_crc(&response[..3 + byte_count + 2]) {
        return Err("crc error".into());
    }

    let mut values = Vec::with_capacity(byte_count / 2);
    for i in 0..(byte_count / 2) {
        let index = 3 + i * 2;
        values.push(u16::from(response[index]) << 8 | u16::from(response[index + 1]));
    }
    Ok(values)
}

fn parse_u8_config(value: Option<&String>, default: u8) -> u8 {
    value
        .and_then(|s| s.trim().parse::<u8>().ok())
        .unwrap_or(default)
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
            let v = s.trim();
            v == "1" || v.eq_ignore_ascii_case("true")
        })
        .unwrap_or(false)
}

fn format_scaled_value(value: f64, decimals: usize) -> String {
    format!("{value:.decimals$}")
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
    let got = u16::from(data[data.len() - 2]) | (u16::from(data[data.len() - 1]) << 8);
    crc16(&data[..data.len() - 2]) == got
}

fn hex_preview(bytes: &[u8]) -> String {
    let mut out = String::new();
    for (index, byte) in bytes.iter().enumerate() {
        if index > 0 {
            out.push(' ');
        }
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
    fn build_read_frame_matches_tinygo_driver() {
        let frame = build_read_frame(0x01, 0x0000, 0x0003);
        assert_eq!(frame, vec![0x01, 0x04, 0x00, 0x00, 0x00, 0x03, 0xB0, 0x0B]);
    }

    #[test]
    fn parse_read_response_matches_expected_points() {
        let mut response = vec![0x01, 0x04, 0x06, 0x00, 0xFD, 0x01, 0xF9, 0x00, 0xBB];
        let crc = crc16(&response);
        response.push((crc & 0x00ff) as u8);
        response.push((crc >> 8) as u8);
        let values = parse_read_response(&response, 0x01).unwrap();
        assert_eq!(values, vec![0x00FD, 0x01F9, 0x00BB]);
    }
}
