# 发布说明

源码仓库为 [lmxx1234567/sushiro-cli](https://github.com/lmxx1234567/sushiro-cli)，npm 包为 [sushiro-cli](https://www.npmjs.com/package/sushiro-cli)。安装方法见 [README](../README.md)，独立二进制和版本产物见 [GitHub Releases](https://github.com/lmxx1234567/sushiro-cli/releases)。

GitHub Release 提供 macOS、Linux、Windows 的 amd64/arm64 六个独立二进制。npm 暂仅提供 macOS/Linux 的 amd64（x64）和 arm64：一个主包与四个平台包，共五包。Windows npm/npx 暂未提供，请使用对应架构的独立二进制。

用户只需安装主包 `sushiro-cli`，npm 自动选择对应平台包，无需手选。npm 启动器运行平台包内的 Go 程序；不使用 postinstall 下载二进制，安装时需保留 optional dependencies。

已配置的工作流由 GitHub Release 的 `published` 事件触发六平台构建、检查和 Release assets；当前启用的五个 npm 包完成首次建立、所有权与 OIDC trusted publishing 配置并开启发布后，才自动发布 npm。首次包建立及 OIDC 未就绪时，不应声称 npm 自动发布已经可用。先发布平台包并核对版本/完整性，再发布主包；部分失败需先核对 registry 状态，不盲目重发。

维护者发布前需核对集成后的 workflow、包名和版本、权限与 OIDC、来源及许可文件、可用的私密安全报告渠道，以及各平台原生运行范围。每个分发包须保留项目 LICENSE、CREDITS、第三方声明和依赖许可全文，并让隐私、安全和免责声明可发现。工具链或依赖改变时重新核对许可覆盖。

每次发布后都需要从实际 registry 干净安装并验证，离线 tarball 安装无法代替这一步。当前验证状态见 [支持范围](limitations.md)。

构建与包结构见 [npm 分发](distribution/README.md)；首次建包、仓库变量、当前启用包集的 OIDC 配置和失败恢复步骤见 [维护者发布指南](distribution/release-plan.md)。各次执行结果见 [发布工作流](https://github.com/lmxx1234567/sushiro-cli/actions/workflows/npm-release.yml)；GitHub 附件与 npm 发布阶段分别判定成功，不能互相替代。
