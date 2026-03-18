# TinyGo Drivers

这里存放基于 TinyGo 构建的 Wasm 驱动。

## 目录约定

- Go 模块根目录：`drvs/tinygo/`
- 模块文件：`drvs/tinygo/go.mod`、`drvs/tinygo/go.sum`
- 共享工具包：`drvs/tinygo/pkg/`
- 站点驱动目录：`drvs/tinygo/<site>/<driver>/`

每个驱动目录通常包含：

- 驱动源码 `*.go`
- 构建脚本 `Makefile`
- 协议说明 `README.md`
- 点位表 `points.xlsx`
- 编译产物 `*.wasm`

## 构建入口

编译全部 TinyGo 驱动：

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/Makefile tinygo
```

编译单个 TinyGo 驱动：

```bash
make -f /Users/mac/workspace/xunji/fsu/drvs/tinygo/陆家嘴社区卫生服务中心/美的空调/Makefile all
```

## 统一读写约定

TinyGo 驱动与 FSU 的接口约定如下：

- 默认执行读操作
- 当 `config.func_name=write` 时，执行写操作
- 一次调用只允许写一个点
- FSU 根据返回点里的 `rw` 属性判断是否展示写入口
- 只要 `rw` 中包含 `W`，该点就应被视为可写

参考写入请求：

```json
{
  "config": {
    "func_name": "write",
    "field_name": "TEMSET",
    "value": "24.0",
    "device_address": "1"
  }
}
```

说明：

- `field_name` 为驱动点表字段名
- `value` 为工程量字符串，不是原始寄存器值
- 驱动内部负责完成字段匹配、数值换算、寄存器写入和响应校验

## 统一 describe/version 约定

TinyGo 驱动除 `handle` 外，还应导出 `describe` 和 `version`。

`describe`：

- 外层结构统一为 `success + data`
- `data` 至少建议包含 `language`、`transport`、`protocol`、`write`
- 可写驱动可额外返回 `writable_fields`

`version`：

- 外层结构统一为 `success + version + productKey + data`
- 顶层字段用于宿主稳定提取版本和产品标识
- `data` 保留同名字段，并允许驱动补充额外说明

当前 TinyGo 侧已经把这套标准收敛到 `drvs/tinygo/pkg/tinydrv/` 公共包中。

## 当前写入支持状态

- `美的空调`：已实现真实单点写入
- `msj`、`共济温湿度`、`ups`、`列头柜`、`青鸟消防`、`高特电池网关`：当前为只读驱动，收到写请求会明确返回不支持
- `压力传感器`、`液位传感器`：当前也按只读驱动使用

## 开发建议

- 新增驱动时，优先把点表定义、寄存器分组、工程量换算写清楚
- 写入能力要和点位 `rw` 标记保持一致，不能只改前端展示
- README 应同时说明读取分组、写入支持范围、编译方式和配置要点
