# TinyGo Drivers

这里存放基于 TinyGo 构建的 Wasm 驱动。

- Go 模块文件位于 `drvs/tinygo/go.mod` 和 `drvs/tinygo/go.sum`
- 站点目录按 `tinygo/<site>/<driver>/` 组织
- 共享 TinyGo 工具包位于 `drvs/tinygo/pkg/`
- 统一构建入口：
  `make -f drvs/Makefile tinygo`
