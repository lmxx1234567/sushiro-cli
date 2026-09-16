# 来源与许可边界

感谢 [Ryujoxys/sushiro-overdose](https://github.com/Ryujoxys/sushiro-overdose)。本项目基于固定 revision `e273df046789773616c7851c0bea14d4546f47e5` 的以下实现，并保留其 MIT 版权和许可；不是完全原创或 clean-room 实现。

| 本地范围 | 固定上游来源及复用方式 |
| --- | --- |
| `internal/api/client.go` | [api.go](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/api/api.go)：改写适配门店/时段/预约请求、认证选择、E044/E052 语义及嵌套票据解析；本地增加响应限制、错误隐藏、校验与不重试语义 |
| `internal/api/types.go` | [slot.go](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/core/slot.go)：Slot 完整模型声明直接复用；门店和预约模型按协议改写缩减 |
| `internal/api/public_defaults.go` 和门店缺省参数 | [queue_live.go](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/app/queue_live.go)：直接沿用公共查询默认 token、非定位坐标和结果数约定；本文不显示 token 值 |
| `internal/auth` | [capture.go](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/core/capture.go) 与 [auth_import.go](https://github.com/Ryujoxys/sushiro-overdose/blob/e273df046789773616c7851c0bea14d4546f47e5/internal/app/auth_import.go)：本地严格导入和分离存储受其凭证/请求头研究启发，未沿用公共与个人认证自动互补 |

CLI/service/MCP 和 npm 启动器采用本地架构，没有引入上游桌面/Web UI、预测采样、调度器或 Python FastMCP。仓库组织方式不消除来源或许可义务。

本项目新增贡献采用 [MIT](../LICENSE)，不替换上游和依赖的版权。完整致谢见 [CREDITS](../CREDITS.md)，上游 MIT 及依赖范围见 [第三方声明](../THIRD_PARTY_NOTICES.md) 和 [许可全文](../THIRD_PARTY_LICENSES.txt)。

MCP Go SDK v1.4.0 含 Apache-2.0/MIT 过渡声明：新或已获重授权贡献适用 Apache-2.0，未获同意重授权的原 MIT 贡献仍保留 MIT；不是单一 MIT，也不是可以任择的双许可。其非规范文档为 CC-BY-4.0；本项目未复制 SDK 文档。其他依赖与 Go 工具链的许可、专利文本按各自范围保留，改变工具链或依赖时须重新核对。

公共默认值进入源码及二进制，缺少覆盖配置时仅用于公共查询，不自动写入用户目录。上游代码采用 MIT 不证明后台服务授权、token 颁发归属、可转移性、分发条件或长期有效性；当前尚未取得运营方对此的确认。功能验证边界见 [支持范围](limitations.md)。
