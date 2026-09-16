# 开发与测试

普通源码构建需要 Go 1.24+；分发构建与验证固定 Go 1.26.5，以匹配已审工具链许可。npm 分发测试另需 Node.js 20+ 和 npm，正式发布工作流固定 Node.js 24.13.0 与 npm 11.6.2。当前源码 module 为 `github.com/lmxx1234567/sushiro-cli`，程序入口为 `./cmd/sushiro-cli`。

```sh
go build -o ./dist/sushiro-cli ./cmd/sushiro-cli
go test -race ./...
go vet ./...
node --test packaging/test/*.test.mjs
```

Go 测试使用 mock HTTP 与临时目录。Node 测试构建六个目标，生成七个私有 npm 包，在隔离目录 pack/install；fixture 测试检查参数、流、退出码和信号，real-cli 测试检查真实程序版本、MCP 初始化/工具发现、空配置与许可。发布策略测试另覆盖版本/事件校验、包内政策链接、归档与重跑规则。测试可能需要先取得工具链或依赖，但默认不请求线上寿司郎业务、不发布 registry、不修改全局安装。

`internal/api` 处理请求与响应；`internal/auth` 管理严格导入和独立存储；`internal/service` 统一结果与写入语义；`internal/cli`、`internal/mcp` 提供入口；`packaging` 和 `scripts/release` 负责分发。公共查询不得加载个人档案或发送个人身份字段。

`packaging/test/public-live.mjs` 是独立线上公共验收脚本，不在默认离线测试中运行。若另行使用，必须先阅读脚本、配置独立空目录并限定公共 GET 范围；不要把真实配置或结果原文提交。日常贡献无需线上调用。当前验证覆盖见 [支持范围](limitations.md)。

遵循 [贡献指南](../CONTRIBUTING.md)，发布步骤见 [发布说明](releases.md)。
