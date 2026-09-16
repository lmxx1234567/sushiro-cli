# 安全说明与漏洞报告

更新日期：2026-09-16。当前处于开源发布准备阶段，尚未确定长期支持版本范围、响应时限或漏洞奖励计划。不要据此假定旧版本有安全维护承诺。

## 如何报告

**发布前待完成：维护者应确认安全联系人，并启用、验证 GitHub 私密漏洞报告或公布可用的私密联系渠道。** 计划发布账号为 `lmxx1234567`、仓库为 [lmxx1234567/sushiro-cli](https://github.com/lmxx1234567/sushiro-cli)，目前尚未创建。当前没有已确认的私密地址，不应把计划入口当成已可用的联系方式。

项目公开后，如果仓库的 Security → Advisories 页面提供 **Report a vulnerability**，请通过该入口报告；它仅在维护者启用后可用。若没有该入口或已公布的私密渠道，可在项目 issue 中只询问“如何私密联系安全维护者”，不附漏洞利用细节、个人数据或凭证；等待确认渠道后再发送详情。这与 [GitHub 官方报告指引](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/report-privately)一致，不代表本仓库已启用该功能。

报告应尽量包含版本/commit、操作系统、CLI 或 MCP 使用方式、影响描述，以及使用虚构账户和假 token 的最小复现。可以提供固定错误码及脱敏截图。不要发送实际 Authorization、WechatID、手机号、票据号码、完整凭证 JSON、代理捕获、请求头或未经检查的宿主日志。私密渠道仍可能由第三方托管，并非可以放心提交全部秘密的理由。

如已泄露个人凭证，应先停止进一步传播、通过官方服务处理会话撤销/更新，并检查真实账户状态。删除本地文件或公开帖子不保证服务端凭证失效，也不能收回所有副本。涉及寿司郎或微信服务本身的问题应同时使用其官方渠道；本项目无权承诺第三方的修复时限。

## 已实现的保护及边界

- 公共查询与个人档案分开；公共客户端限制为公共路径的 GET。显式损坏或不安全公共配置不会静默退回默认配置。
- 业务目标固定为寿司郎 HTTPS origin，不跟随 HTTP 重定向。默认网络层仍受系统信任和环境代理配置影响。
- Unix 配置目录/文件创建权限为 `0700`/`0600`，包含路径和文件检查；保存为未加密 JSON，同用户应用和管理员仍可能读取。Windows 私有文件存储尚不支持。
- 业务错误不直接回传原始响应、请求 URL 或头。正常票据输出、交互确认中的操作参数，以及系统、终端或 MCP 宿主保存的内容仍需人工脱敏。
- 写操作要求确认标志，服务层没有自动重试写请求的循环；不确定结果尝试只读回查，不把未确认状态当作成功。确认标志、MCP annotations 都不是独立的授权或安全隔离机制。

这些是代码层面的保护，不是对不存在漏洞、不会泄露或不会发生未经授权操作的保证。源码中的公共默认 token 是公开分发内容，不能当作个人账户凭证或秘密保护措施，其分发条件和有效期仍需确认。

## 使用和维护建议

只向可信的 MCP 宿主开放所需工具，核对每次写操作的具体参数。避免将个人工具开放给不可信对话或自动流程。不要将原始凭证放进命令行、环境变量、聊天、issue 或仓库；配置目录的环境变量只用于路径。使用私有文件导入并妥善处理导入源文件和备份。

只使用来源可核对的发行物；发布文件哈希有助于检查一致性，但不能独立证明发布者身份。贡献者应使用虚构数据和测试 transport 复现问题，不要为测试实际预约或取消。不要在未经单独授权时运行 `docs/auth/observer/` 研究原型、修改系统代理或安装证书。

具体数据流、保留和删除方式见 [PRIVACY.md](PRIVACY.md)，配置方法见[配置说明](docs/configuration.md)，功能边界见[已知限制](docs/limitations.md)。

## English summary

A working private security contact must be confirmed before release. Use GitHub's “Report a vulnerability” only if enabled; otherwise ask for a private contact without posting sensitive details. Reports should use synthetic credentials and minimal reproductions. No response-time, support-lifetime or bounty commitment has been established. Local credentials are unencrypted, MCP hosts may process personal outputs, and confirmation flags do not prove user authorization. Do not disclose real tokens or run account writes to reproduce a security issue.
