# 发布说明

当前尚未公开 npm 包；预定源码仓库为 `lmxx1234567/sushiro-cli`。在实际发布前使用 [源码构建](../README.md)，不要将示例包名当作可安装的已发布包。

目标产物为 macOS、Linux、Windows 的 amd64/arm64 二进制，以及一个 npm 主包和六个平台包。npm 启动器运行平台包内的 Go 程序；不使用 postinstall 下载二进制，安装时需保留 optional dependencies。

已配置的工作流由 GitHub Release 的 `published` 事件触发六平台构建、检查和 Release assets；七个 npm 包完成首次建立、所有权与 OIDC trusted publishing 配置后自动发布。首次包建立及 OIDC 未就绪时，不应声称 npm 自动发布已经可用。先发布平台包并核对版本/完整性，再发布主包；部分失败需先核对 registry 状态，不盲目重发。

维护者发布前需核对集成后的 workflow、包名和版本、权限与 OIDC、来源及许可文件、可用的私密安全报告渠道，以及各平台原生运行范围。每个分发包须保留项目 LICENSE、CREDITS、第三方声明和依赖许可全文，并让隐私、安全和免责声明可发现。工具链或依赖改变时重新核对许可覆盖。

发布后需要从实际 registry 干净安装并验证，离线 tarball 安装无法代替这一步。当前验证状态见 [支持范围](limitations.md)。

构建与包结构见 [npm 分发](distribution/README.md)；首次建包、仓库变量、七包 OIDC 配置和失败恢复步骤见 [维护者发布指南](distribution/release-plan.md)。工作流已提供不代表已在远端执行或完成发布。
