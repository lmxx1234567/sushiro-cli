# Go CLI 的 npm 分发

实现使用 Node 内置模块，无 npm 开发依赖。npm 只分发预编译 Go 程序；不使用 postinstall、不在用户安装阶段从外部下载二进制。生成的主包提供 `sushiro-cli` 命令，所有 CLI 参数（包括 `mcp`）原样传给 Go 程序。

## npm 平台支持

0.1.3 的 npm 发布目标为 macOS/Linux（x64、ARM64）；Windows npm 暂不可用，Windows 用户可使用 GitHub Release 独立二进制。主包通过 `os` 限制安装平台。维护者可用 `npmTargets` 显式选择平台；省略时保持历史六平台行为，空数组或无效目标不会静默降级。详见 [发布配置](release-plan.md)。

## 安装后的首次使用

安装与启动不要求导入个人凭证。安装主包并保留 optional dependencies 后，直接使用 `sushiro-cli help`、`sushiro-cli version --json`，或将 MCP 客户端配置为执行 `sushiro-cli mcp`；MCP 初始化和工具发现不需要个人登录。npm 启动器不读取个人凭证，也不以 `auth status.complete` 作为启动条件。

公共 `stores` / `slots` 查询与个人认证分开。缺少独立公共配置文件时使用内存中的上游默认查询配置，不读取个人档案、不自动创建配置文件；这仍是携带查询授权的公共请求，不能称为完全匿名或个人登录成功。`sushiro-cli public status` 标明 `configuration_source=builtin_default`、`persisted=false`，不会输出值。

显式公共配置优先；坏文件、权限不安全或符号链接报 `public_config_invalid`，显式空查询字段报 `public_config_required`，都不会静默回退。需要更换默认配置时按 [公共配置说明](../configuration.md) 准备自己的受限文件。个人微信登录或个人凭证导入仍不是公共查询前置。


`ticket-status`、`reservations`、`reserve`、`cancel` 等原有功能仍保留。需要个人功能时再按对应要求准备私有凭证，写操作另需具体参数授权。npm 不自动触发登录或凭证导入。

## 卸载与配置保留

在原安装位置运行 `npm uninstall <scope>/<name>`（仅当原本是全局安装时才加 `-g`）移除 npm 包。npm 卸载不会删除用户的独立配置，本包也没有自动清理配置的卸载脚本。

确需清理配置时，应先核对实际使用的目录，再由用户明确删除目标文件或目录：

- 个人配置根目录优先取 `SUSHIRO_CONFIG_DIR`；默认 macOS 为 `~/Library/Application Support/sushiro`，Linux 为 `${XDG_CONFIG_HOME:-~/.config}/sushiro`。
- 公共配置目录优先取 `SUSHIRO_PUBLIC_CONFIG_DIR`；未设置时为上述配置根目录内的 `public` 子目录。
- 只清理某个档案时，对应文件为选定目录中的 `<profile>.json`。清理整个个人根目录也会连带删除默认位置的公共配置，应先确认范围。

上述环境变量只保存路径，不能放入 token。删除磁盘配置也不等于使服务端令牌失效。

## 配置及构建

发布构建固定 Go **1.26.5**，匹配工具链许可审计；安装后的 Node 启动器支持 Node >=20。将 `packaging/npm/config.example.json` 复制到自己的配置路径，修改 `scope` 和可选的 `name`。示例 scope 是占位符，无 scope 可设空字符串；发布者应先确定包名及 scope 权限。主项目采用 MIT 并保留第三方条款，所有默认准备产物仍为 `private: true`。

```sh
node scripts/release/build.mjs --version 0.1.0
node scripts/release/prepare.mjs --config /absolute/path/npm-config.json --version 0.1.0
```

构建入口严格为 `./cmd/sushiro-cli`，CGO 关闭。默认 dist 为 `packaging/out/dist`，可用 `--dist /absolute/path/dist` 更改；prepare 必须使用同一个 dist。产物遵循 `dist/<goos>-<goarch>/sushiro-cli[.exe]`。构建生成 `build.json`，记录版本、源码提交、工作区是否干净、Go 版本、六个二进制的 SHA-256。Go 最低 1.24.0。构建通过 `-ldflags "-s -w -X main.version=<version>"` 注入版本，使 CLI `version --json` 与包版本一致。

prepare 默认输出 `packaging/out/npm`，也支持 `--out`。已有输出目录会报错，避免混入旧版本；重跑请选择新目录。输出包括六个平台素材目录、主包目录、所选平台加主包的 `.tgz`、`SHA256SUMS` 与 `release.json`。其中 `release.json` 列出 tarball 路径、SHA-256、npm integrity、包内文件列表和构建记录。

| Go 目录 | npm 平台包后缀 | 程序 |
| --- | --- | --- |
| darwin-amd64 | darwin-x64 | sushiro-cli |
| darwin-arm64 | darwin-arm64 | sushiro-cli |
| linux-amd64 | linux-x64 | sushiro-cli |
| linux-arm64 | linux-arm64 | sushiro-cli |
| windows-amd64 | win32-x64 | sushiro-cli.exe |
| windows-arm64 | win32-arm64 | sushiro-cli.exe |

有 scope 时主包名为 `<scope>/<name>`，无 scope 时为 `<name>`；平台包在主包名后加 `-<os>-<cpu>`。所选平台的 `optionalDependencies` 使用同一个精确版本，不带 `^` 或 `~`。平台包声明 `os` / `cpu`，由 npm 选择；启动器还会检查实际安装包版本。依据 [npm package.json 文档](https://docs.npmjs.com/cli/v11/configuring-npm/package-json/)，平台筛选和可选依赖是安装行为，安装时不能省略 optional dependencies。

## 启动器行为

- 通过无 shell 的 spawn 运行本机平台包内二进制，继承 cwd、环境及 stdin/stdout/stderr；启动成功不输出任何 banner。
- Go 普通退出码原样返回。POSIX 下向启动器发出的 SIGINT、SIGTERM、SIGHUP 转发给子进程；子进程因信号退出时启动器也以该信号终止。
- Windows 的 Node 信号是有限的终止模拟，不能承诺 POSIX 的可捕获信号语义。Windows 控制台 Ctrl+C 和 MCP 客户端终止行为需要原生环境验收。
- 缺少依赖、版本不一致、不支持的平台、不可执行二进制均只在 stderr 报错并返回 1，不污染 MCP stdout。
- SIGKILL 无法被拦截，因此不能保证父进程被强杀后的子进程清理。程序应处理 stdin EOF；这属于 Go/MCP 集成验收。

## 本地测试

```sh
node --test packaging/test/distribution.test.mjs
```

测试在系统临时目录编译明确标注的 fixture，为全部六个目标生成原生二进制，再 pack 七个包。在隔离项目中，用本地 tarball 覆盖全部平台依赖，`npm install --offline --ignore-scripts`；检查只安装本机平台。离线安装使用独立 cache 和空 npm 配置，不访问 registry，不修改全局安装。成功后清理临时目录，失败保留路径以便排查。

测试覆盖 npm 命令 shim、带空格路径、Unicode/空参数、stdin/stdout/stderr、MCP 字节透传、0/7/130/255 退出码、POSIX 三种信号、信号退出状态，以及缺包/错版本/权限错误。fixture 的 `mcp` 只是回显输入，不证明 MCP 协议实现正确。`packaging/test/fixture.go` 绝不可当正式 CLI。

若使用仅带 Node 的运行时，可将 `NPM_CLI` 设为本地 npm 安装的 `bin/npm-cli.js`；这不是下载器。Windows 脚本优先发现 Node 同目录的 npm，找不到时需要此配置。

`.github/workflows/distribution-test.yml` 提供 macOS/Linux/Windows 原生包装层测试任务；某个 runner 仅验证其实际 CPU 架构。工作流尚未运行不能视为这些平台已验收。

## 发布准备与顺序

`distribution-prepare.yml` 只手动触发，使用真实核心构建，上传带 `private: true` 的审核产物，没有 npm 凭证或 publish 步骤。构建失败时不会使用 fixture 替代。

自动流程为 `npm-release.yml`：GitHub **Release published** 后执行跨平台构建、测试与附件准备；完成首次所选包建包和 OIDC 设置并显式开启后，平台包先发布，主包最后。未开启时 summary 明确 npm 未发布。GitHub Release 不代表 npm 已可安装；包名、账号检查、bootstrap、OIDC配置及失败重跑规则见 [发布指南](release-plan.md)。

正式发布前，确认 scope 权限、包名、版本及第三方归属。`license: "MIT"` 仅代表主项目许可。每个包的 `legal/` 必须包含 LICENSE、CREDITS.md、THIRD_PARTY_NOTICES.md、THIRD_PARTY_LICENSES.txt、PRIVACY.md、DISCLAIMER.md、SECURITY.md；若存在 NOTICE/NOTICE.md 也包含。`legalFiles` 可追加经审阅的材料（路径相对于配置文件）；不复制完整 docs 目录。

从干净的已提交真实源码运行 build，再选择新的输出路径：

```sh
node scripts/release/prepare.mjs --config /absolute/path/npm-config.json --version 0.1.0 --out packaging/out/release-0.1.0 --publish-ready
```

此模式要求已选定有效包名、最终法律/隐私文件、已审 Go1.26.5 工具链、干净真实 CLI 构建记录、精确版本及六个文件哈希一致，并只读确认 GitHub 仓库和该构建 commit 已公开，才将 `private` 设置为 false。构建记录用于追溯，不是防篡改签名；发布者仍须审核来源、tarball 文件清单和运行验收。

经单独授权发布时，严格按 `release.json` 中的 packages 顺序：先六个平台包，确认 registry 中的版本/完整性可用，再发布最后的主包。每个包使用本次审核的 tarball，例如 `npm publish <tarball> --access public`；预发布版本另加合适的 `--tag`，不能误用 latest。发布后需在干净环境从 registry 安装验证实际依赖解析。遇到部分发布失败，先查询已存在版本并核对完整性，不直接重发或替换产物。

## 集成验证与剩余验收

`node --test packaging/test/real-cli.test.mjs` 构建真实核心并执行私有离线 pack/install，通过 npm 安装生成的命令入口，在独立空配置目录核对 help、version、JSON、MCP 初始化/六个工具发现/EOF/终止信号及许可证内容。非 Windows 环境另检查 public status 的内存默认来源及不落盘，并生成无秘密的坏 JSON 验证不静默回退。测试不调用任何业务工具或线上 API，也不读取已有个人配置，并确认结束后两个配置目录仍为空。六目标交叉编译与本机原生运行分别记录。

需要继续用同一安装执行已授权的公共线上验收时，可设置非秘密开关 `SUSHIRO_KEEP_TEST_ARTIFACTS=1` 保留测试产物，测试将输出临时目录。另行运行：

```sh
node packaging/test/public-live.mjs --install <临时目录>/install --config-dir <空个人配置目录> --public-config-dir <独立公共配置目录绝对路径> --mode configured --out <报告路径>
```

它仅通过 npm 安装的命令入口调用公共 stores（固定测试坐标 `39.97,116.43`、limit=2，并非用户定位）、3006 单店和 2 成人 T 桌 slots，并分别执行对应 MCP 查询。它会访问线上，故不属于默认离线测试；报告只含结果状态、数量及错误码，不保存原始响应。

`--mode configured` 要求该公共目录存在 `default.json`；`--mode default` 要求该公共目录为空，专门验收内存默认查询路径。报告区分模式和构建提交，并检查查询前后公共目录条目一致、个人目录仍为空；默认模式不会导出内置值供检查。

脚本清除继承的 `SUSHIRO_*` 后，仅显式指定空个人配置目录与独立公共配置目录；不读取已有个人档案、不接受 token 参数、不导入或复制公共配置，也不会尝试修复或绕过上游查询授权。成功要求门店列表非空且最多两家、单店 id 精确为 3006、时段非空且 storeId/date/start/end/availability 均为非空字符串、storeId=3006，并验证 YYYYMMDD 日期及 HHMMSS 时间有效。列表为空不是本次可用性验收成功；特殊坐标可能合法返回空列表，应另行说明其业务语义。业务成功必须以报告的 `passed` 及各项实际结果为准。

发布前还需目标系统原生验收，特别是 Windows 控制台终止行为；安全凭证存储限制见 [功能限制](../limitations.md)。交叉编译不等于目标系统运行通过。
