# sushiro-cli

面向寿司郎中国大陆的非官方 Go 命令行工具，同一程序可运行 stdio MCP。当前主要支持无需个人登录的门店、等位桌数与预约时段查询。

感谢 [Ryujoxys/sushiro-overdose](https://github.com/Ryujoxys/sushiro-overdose) 的开源工作：本项目改写适配其 API 请求和解析，直接复用 Slot 模型及公共查询默认值。具体来源见 [CREDITS](CREDITS.md)、[来源说明](docs/provenance.md) 和 [第三方声明](THIRD_PARTY_NOTICES.md)。本项目与寿司郎、微信及上游作者不存在官方背书关系。

个人查询与预约写操作为实验性功能，尚未完成真实业务闭环；不提供原生微信登录。

## 安装

npm 安装仅支持 macOS/Linux 的 amd64（x64）和 arm64，需要 Node.js 20+ 和 npm。公开包为 [sushiro-cli](https://www.npmjs.com/package/sushiro-cli)，源码与独立二进制见 [GitHub](https://github.com/lmxx1234567/sushiro-cli) 和 [Releases](https://github.com/lmxx1234567/sushiro-cli/releases)。

```sh
npm install -g sushiro-cli
sushiro-cli version --json
sushiro-cli help
```

也可无需全局安装，通过 npx 运行：

```sh
npx --yes sushiro-cli@0.1.3 help
npx --yes sushiro-cli@0.1.3 stores --near 39.97,116.43 --limit 2 --json
```

只安装主包 `sushiro-cli`，npm 会自动选择对应平台包，无需手选。请保留 optional dependencies；首次 npx 运行可能下载包。安装不要求个人登录。MCP 的 npx 配置见 [MCP 文档](docs/mcp.md)。

Windows 暂不提供 npm/npx 安装。请从 [GitHub Release v0.1.3](https://github.com/lmxx1234567/sushiro-cli/releases/tag/v0.1.3) 下载与 CPU 架构对应的 Windows 独立二进制，解压后运行 `sushiro-cli.exe`；其个人凭证安全存储仍未实现。

## 从源码运行

需要 Go 1.24 或更新版本。在源码目录运行：

```sh
go build -o ./dist/sushiro-cli ./cmd/sushiro-cli
./dist/sushiro-cli help
./dist/sushiro-cli version --json
./dist/sushiro-cli stores --near 39.97,116.43 --limit 2 --json
./dist/sushiro-cli stores --store 3006 --json
./dist/sushiro-cli slots --store 3006 --adult 2 --child 0 --table T --json
```

Windows 构建请将输出命名为 `sushiro-cli.exe`。查询命令会访问寿司郎服务；help、version 与本地配置状态检查不需要个人会话。

`--near` 是自行指定的参照坐标，不会获取设备定位；省略时使用兼容缺省 `1,1`。`wait` 表示等位桌数，不是分钟。查询到时段不保证提交预约时仍有名额。

公共查询使用随源码和二进制分发的上游默认查询配置，不读取个人档案，也不自动创建配置文件。无需个人登录不等于匿名或无 Authorization；默认值的有效期、分发条件与官方服务授权尚未确证。

## 文档

- [使用与输出](docs/usage.md)：CLI 命令、错误和写操作边界。
- [配置](docs/configuration.md)：独立公共配置、个人导入与卸载保留。
- [MCP](docs/mcp.md)：客户端配置和工具参数。
- [支持范围与验证](docs/limitations.md)：已验证能力与未完成项。
- [开发与测试](docs/development.md)、[贡献指南](CONTRIBUTING.md)。
- [发布说明](docs/releases.md)：构建、分发和发布前条件。

新增贡献采用 [MIT](LICENSE)；分发需保留 [CREDITS](CREDITS.md)、[第三方声明](THIRD_PARTY_NOTICES.md) 与 [依赖许可全文](THIRD_PARTY_LICENSES.txt)，不将混合依赖二进制描述为纯 MIT。使用前请阅读 [隐私说明](PRIVACY.md)、[免责声明](DISCLAIMER.md)；安全反馈见 [SECURITY](SECURITY.md)。
