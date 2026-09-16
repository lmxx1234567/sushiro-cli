# 隐私与数据处理说明

更新日期：2026-09-16。适用于本仓库的 `sushiro-cli` CLI、stdio MCP 及 npm 启动器；使用配置见[配置说明](docs/configuration.md)，功能边界见[已知限制](docs/limitations.md)。发布账号/署名为 `lmxx1234567`；其法律主体身份、隐私联系安排和适用法域尚待确认，不能仅从 GitHub 用户名推定。本文不构成已经完成法律合规审查的声明。

## 数据流

当前产品代码没有维护者运营的数据中转、使用分析、遥测或崩溃上报入口。业务请求以 HTTPS 发往 `crm-cn-prd.sushiro.com.cn`，不会先交给本项目维护者的服务器。此结论限定于核对过的产品代码，不涵盖自行修改版本、包管理器、MCP 宿主或另行运行的研究脚本。

“公共查询”表示无需你的个人业务会话，并不表示匿名或无认证。默认公共查询 token 随源码和二进制分发，只在运行时作为内存默认配置使用，不自动写入用户配置目录。它的签发主体、分发条件和有效期尚未确证；来自公开上游源码不等于获得寿司郎官方授权。

| 功能 | 发给寿司郎接口的数据 | 本地输出 |
| --- | --- | --- |
| 门店列表、单店查询 | 公共查询 Authorization、客户端头；列表坐标与数量，或门店 ID | 门店名称、地址、区域、营业/预约状态、等位桌数等选定字段 |
| 可约时段 | 公共查询 Authorization、客户端头、门店 ID、桌型、合计人数 | 门店、日期、起止时间、可用状态 |
| 当前票据 | 个人 Authorization、客户端头、URL 查询参数中的 WechatID | 当前排队/预约票据的选定字段，可能包括票号、门店、日期时间、人数 |
| 完整预约列表 | 个人 Authorization、客户端头、WechatID、手机号 | 预约记录的选定字段，或明确的失败状态 |
| 创建或取消预约 | 个人 Authorization、客户端头、WechatID、手机号，以及预约参数或票据 ID | 票据、操作状态；必要时包含只读回查结果 |

客户端头可包括 User-Agent、Referer、X-App-Code、X-App-Client；个人请求另带 Xweb_xhr。是否发送某个可选头取决于配置。公共查询不加载个人档案，也不发送其中的 WechatID 或手机号。

程序不自动获取设备定位。列表未指定 `--near` / `near` 时使用兼容坐标 `1,1`，不是你的位置；显式提供的坐标会发送给服务端。寿司郎服务端能看到请求来源 IP（使用代理时通常是代理出口 IP）、请求时间、请求头和参数。项目无法规定该服务端的留存、共享或删除方式，应查阅你使用的官方服务内的现行隐私说明。

网络路径还受你的环境影响：Go 默认 HTTP transport 可使用 `HTTPS_PROXY` 等代理配置，DNS、代理及受信任的网络检查设施也可能处理连接信息。“目标是寿司郎接口”不等于网络上没有其他参与方。[Go 官方说明](https://pkg.go.dev/net/http#ProxyFromEnvironment)

## 本机保存、保留与删除

个人模式通过 `auth import --file` 导入已有会话；原生微信登录尚未完成。保存内容包括个人授权 token、WechatID、手机号及兼容请求头，具体以导入文件为准。`public import --file` 保存单独的公共覆盖配置，其 schema 拒绝个人身份字段。导入不会自动删除原始文件。

在支持的 Unix 系统上，程序创建配置目录时使用 `0700`、写入文件时使用 `0600`，并拒绝部分不安全路径/权限。文件是**未加密 JSON**，不是系统钥匙串或凭证保险库。同一用户权限的应用、管理员、恶意软件以及有权限的备份程序仍可能读取。已有的更严格权限可以保留；权限检查不能保证所有文件系统、ACL 或并发攻击场景都安全。Windows 当前不支持这些私有文件存储操作；内置默认公共查询不依赖该存储能力。

| 系统 | 默认个人档案路径；公共档案位于其 `public/` 子目录 |
| --- | --- |
| macOS | `~/Library/Application Support/sushiro/<profile>.json` |
| Linux | `$XDG_CONFIG_HOME/sushiro/<profile>.json`；未设置时 `~/.config/sushiro/<profile>.json` |

`SUSHIRO_CONFIG_DIR` 可指定配置根目录，`SUSHIRO_PUBLIC_CONFIG_DIR` 可单独指定公共目录；这些变量仅应放路径，不要放 token。默认 profile 是 `default`。`auth status` 只检查本地字段存在情况，可能创建配置目录；它不证明服务端会话有效。公共默认查询和 `public status` 在无覆盖文件时不创建默认配置文件或目录。

显式导入的数据没有自动到期删除机制，升级和卸载 npm 包后仍保留在用户配置目录；本版没有 CLI 清理命令。需要删除时，先停止 CLI/MCP，确认实际覆盖路径及所选 profile，再手动删除对应个人或公共 JSON 文件，并分别处理原始导入文件、备份和自己保存的输出。删除公共覆盖文件会恢复内置默认查询行为；删除个人文件不会撤销寿司郎服务端会话，也不会取消预约。文件删除不保证安全擦除。npm 自第 7 版起没有卸载生命周期脚本，不能依靠包卸载清理这些外部文件。[npm 官方说明](https://docs.npmjs.com/cli/v11/using-npm/scripts/#a-note-on-a-lack-of-npm-uninstall-scripts)

## 输出、MCP 与第三方

CLI 将结果写到标准输出，MCP 通过 stdin/stdout 向宿主返回文本及结构化结果。产品不自动保存业务历史或完整 HTTP 日志；业务错误信息按固定类别返回，不直接转储请求头、URL 或原始响应。但正常票据结果本身可能具有个人性，输出重定向、终端记录、shell 历史、MCP 会话记录和系统诊断可能产生副本。交互写操作确认会展示已校验的操作参数；系统或第三方错误信息还可能包含本机路径等环境信息。不要把凭证粘贴到命令参数或交互窗口，不要把未经检查的日志视为已脱敏。

MCP 宿主能够接收工具参数和输出，并可能将其保存、发送给模型供应商或其他服务。是否用于训练、保存多久、在哪个地区处理，取决于所选宿主、供应商、账户类型和设置，本项目不能保证“永不离机”或“不会被用于训练”。工具的只读/写入提示不是宿主权限控制；请按需要限制个人工具与写操作。

本 CLI/MCP 产品没有桌面截图、Computer Use、浏览器控制、系统代理设置或证书安装能力。若宿主另外执行 Computer Use，其截图和桌面内容由宿主按自身权限处理，不是本 CLI 的采集行为。开发阶段另有 `docs/auth/observer/` 凭证观察研究原型，可捕获会话，其协调脚本涉及代理和证书；它不由 CLI/npm 自动启动，计划不纳入首发公开文件。不能把本产品的能力说明套用于主动运行该原型。

从 GitHub/npm 下载、访问项目页面、提交 issue 或发送附件，会另外与这些第三方交互。维护者可看到你主动提供的反馈，公开 issue 的内容还可能被他人复制；“无产品遥测”不表示维护者不可能收到你提交的数据。相关平台按其政策处理账户、网络和内容信息：[GitHub 隐私声明](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement)、[npm 隐私说明](https://docs.npmjs.com/policies/privacy/)。

计划项目地址为 [lmxx1234567/sushiro-cli](https://github.com/lmxx1234567/sushiro-cli)，目前尚未创建。发布后，非敏感反馈可通过项目 issues 提交，敏感问题仅使用经确认启用的私密渠道；这不是已启用联系方式的声明。

发现误发凭证或隐私问题时，停止继续分享，按[安全说明](SECURITY.md)联系维护者；涉及官方会话失效、撤销或官方数据请求，应使用寿司郎/微信现行官方渠道。维护者不能代替第三方承诺删除其全部副本。

## English summary

The reviewed CLI/MCP code has no maintainer telemetry or relay backend. Business requests target Sushiro over HTTPS, subject to your proxy settings. Public queries use an embedded query token and disclose request metadata and supplied coordinates; they are not anonymous. Imported personal credentials are unencrypted local JSON protected by Unix permissions, readable by applications running as the same user. Imported files survive uninstall. MCP hosts and model providers may receive, store or process tool results. GitHub/npm and voluntarily submitted reports have separate data flows. Publisher identity, privacy contact and applicable jurisdiction remain to be confirmed before release.
