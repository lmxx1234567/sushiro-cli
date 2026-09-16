# sushiro-cli 发布指南

本指南面向维护者：先公开 GitHub 源码，再准备 npm 发布。GitHub owner 为 `lmxx1234567`；GitHub 身份不证明同名 npm 账号或 scope 的所有权。

## 命名和只读查询

Go module：`github.com/lmxx1234567/sushiro-cli`；构建入口：`./cmd/sushiro-cli`；原生程序、npm 命令与 MCP 名均为 `sushiro-cli`。配置目录 `sushiro`、`SUSHIRO_CONFIG_DIR` / `SUSHIRO_PUBLIC_CONFIG_DIR` 保持原有含义。

使用 `node scripts/release/check-npm.mjs --out <报告路径>` 检查候选包名和当前 npm 身份。脚本只读，不打印认证配置或 token。registry 返回404仅表示没有可见公开包，不能保证名称可注册；`npm whoami` 失败也不影响离线构建。

在发布前选定 npm 主包名：无 scope 时配置 `scope: ""`，候选 `sushiro-cli`；有 scope 时配置 `scope: "@lmxx1234567"`，候选 `@lmxx1234567/sushiro-cli`。无论包名选择哪种，安装后的命令都为 `sushiro-cli`。六个平台包加后缀 `-darwin-x64`、`-darwin-arm64`、`-linux-x64`、`-linux-arm64`、`-win32-x64`、`-win32-arm64`。

## 0.1.3 macOS/Linux npm 首发配置

Windows npm 包目前因 registry 名称审核阻塞而暂不可用；Windows 用户使用 GitHub Release 的独立二进制。0.1.3 使用四个平台包加主包，共五包，不能修改或重发已发布的 0.1.2 产物。

本地配置设 `npmTargets: ["darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64"]`。发布前将仓库变量 `NPM_TARGETS` 设为同一个 JSON 数组，`NPM_PACKAGE_NAME=sushiro-cli`、`NPM_SCOPE` 留空。这是明确的平台选择，非法值、重复值和空数组会失败；不设置此字段时兼容历史六个平台。

prepare 始终验证全部六个构建目标，并保留六个平台的独立二进制素材，但只打包显式选择的 npm 依赖。主包声明 `os: ["darwin", "linux"]`，精确依赖四个同版本平台包。`release.json.npmTargets` 固定该次发布的集合，后续附件、bootstrap 与 publish 均以它核对包集；缺包、多包或没有声明却少包都会失败。修改仓库变量不会改变已生成的 release.json。

本次 npm 顺序为 darwin-x64 → darwin-arm64 → linux-x64 → linux-arm64 → 主包，全部版本 0.1.3。先审核五个 tarball，主包不存在时完成手工 bootstrap，再为五个包逐一配置 OIDC。Windows npm 恢复必须作为新版本显式加入目标集合并重新验收。

## GitHub Release 自动流程

工作流 `.github/workflows/npm-release.yml` 仅监听 **release.published**，不是 tag push。正式和预发布 Release 都走此入口；普通 tag push 不发布 npm。发布事件的 tag、实际 checkout 的 commit、build.json、所选包的版本必须一致。版本支持 `v1.2.3` / `1.2.3` 和 SemVer 预发布，不接受 build metadata；预发布标记须与版本里的预发布段一致。

1. GitHub-hosted macOS、Linux、Windows runner 运行 Go race/vet 和离线包装测试，不调用寿司郎线上接口。
2. Linux runner 以 `CGO_ENABLED=0` 交叉构建 darwin/linux/windows × amd64/arm64。锁定 Go **1.26.5**，与已审工具链许可证一致；不把 go.mod 的最低 Go 1.24 当作发布工具链。
3. 验证 GitHub 仓库已公开，且该构建 commit 能在该公开仓库访问；法律与隐私文件齐全、源码干净后生成所选精确版本 npm tarball。
4. 生成六个独立二进制 `.tar.gz`、所选 npm `.tgz`、构建/包清单及 SHA256SUMS，保存到该 GitHub Release。每个二进制包和 npm 包包含必要法律/隐私材料。
5. 仅 `NPM_PUBLISH_ENABLED=true` 时进入名为 `npm` 的 GitHub environment，通过 OIDC 顺序发布所选平台包，主包最后。正式版 dist-tag 为 `latest`，预发布固定为 `next`。
6. summary 明确报告测试、构建、附件及 npm 各阶段结果。未开启 npm 时显示 **npm NOT published**；GitHub Release 存在不代表 npm 已可安装。

工作流使用只读权限作为默认值；仅附件 job 有 `contents: write`，仅 npm job 有 `id-token: write`，不设置长期 npm token。Node 固定 **24.13.0**、npm 固定 **11.6.2**。五个 Actions 均固定经官方仓库 tag 核对的40位 commit，workflow 注释保留版本标签。所有版本使用同一 concurrency group，避免多个 Release 同时争用 latest。

## 首发与 OIDC 配置

官方 trusted publishing 要求 npm CLI >=11.5.1、Node >=22.14.0，当前支持 GitHub-hosted runner；配置入口在每个 npm 包的 Settings。新 Trusted Publisher 默认可进行 staged publish，使用本流程直接 `npm publish` 时，必须额外允许该动作。[npm 官方 trusted publishing 文档](https://docs.npmjs.com/trusted-publishers/)

对于尚不存在的所选包，先创建包再逐一配置 Trusted Publisher：

1. 先完成 GitHub 公开源码审核、MIT/第三方许可与隐私审阅，并确认最终 npm 名、账号/scope 权限和首发版本。
2. GitHub 开源后，配置 repo variables：`NPM_PACKAGE_NAME=sushiro-cli`、选定的 `NPM_SCOPE`（无 scope 留空）。保持 `NPM_PUBLISH_ENABLED` 未设置或 false。
3. 创建首个经批准的 GitHub Release（可先选择 SemVer 预发布版本）。自动测试、跨平台构建、生成附件；npm 阶段明确跳过。
4. 下载该 Release 的 `release.json`、`event.json` 和全部所选 `.tgz`。运行 `node scripts/release/bootstrap-plan.mjs --dir <下载目录>`：只核对 SHA256 并输出对应发布 argv，不登录、不执行命令。审核版本、source_commit、npm_tag 和每个哈希。
5. 用户以实际 npm 发布账号完成交互认证/2FA，单独批准后按输出顺序执行**同一套已审 tarball**：所选平台先，主包最后；不要临时生成空壳占名、修改依赖版本或把预发布改成 latest。首次包创建走这一次人工步骤，不向仓库或 Actions secrets 添加长期自动化 token。
6. 对所选包分别在 npm Settings 配置 Trusted Publisher：provider=GitHub Actions，user=`lmxx1234567`，repository=`sushiro-cli`，workflow filename=`npm-release.yml`（不带目录），environment=`npm`，允许直接 `npm publish`。
7. 配置 GitHub environment `npm`（可添加人工审核及 tag 规则），确认所选包/权限/发布者一致后，显式设置 `NPM_PUBLISH_ENABLED=true`。后续 Release published 才自动发布。OIDC 的身份在 publish 时交换，不以 CI 的 `npm whoami` 作为其成功判据。

最终包集以 release.json 为准，平台按 darwin-x64 → darwin-arm64 → linux-x64 → linux-arm64 → win32-x64 → win32-arm64 的顺序筛选，主包最后。

## 重跑、部分成功和不一致

- 每次开始写 npm 前先只读检查全部所选包。包尚不存在时明确报 bootstrap required；存在同一版本时，远端 `dist.integrity` 必须与本次 tarball 完全一致才能跳过，不重新发布、不修改其 dist-tag。
- 所有本地 tarball 再核对 SHA256 与 SHA512 integrity。发现任一同版本内容不同，整次预检失败，不能通过覆盖、unpublish 或替换文件复用版本号。
- 新版本不允许把 latest 向更低稳定版回退；预发布只用 next。每个新 publish 后检查 registry 中相同版本的完整性，确认后才进入下一个包。
- 发布失败或结果不确定时立即停止，不自动重试写入。检查 registry，确认已存在版本/完整性后再重跑。主包在最后，平台包未齐全不会先发布主包。
- GitHub 已有同名附件必须逐字节 SHA256 相同才复用；不同内容拒绝，不使用 `--clobber`。构建 metadata 去除易变检查时间，归档固定时间/属主/顺序，便于重跑比较。
- GitHub workflow artifact 用 run ID + attempt 唯一命名，并通过 build job 输出的 artifact ID 下载；失败 job 重跑复用对应构建，不误取其他版本。

本地测试覆盖发布决策和离线打包；实际发布仍须核对 hosted runner、OIDC、Release 附件和 registry 安装结果。

## 随包材料与边界

每个 npm 主/平台包及独立二进制压缩包包含 `LICENSE`、`CREDITS.md`、`THIRD_PARTY_NOTICES.md`、`THIRD_PARTY_LICENSES.txt`、`PRIVACY.md`、`DISCLAIMER.md`、`SECURITY.md`；若今后有 NOTICE/NOTICE.md，也自动加入。

npm license 字段使用本项目 MIT，不覆盖 Go runtime/vendor、MCP SDK 混合许可及其他依赖。工具链许可审计固定为 Go1.26.5；换工具链先更新审计与完整许可材料，再改版本约束。

npm 包采用明确文件白名单，仅 manifest、生成README、对应二进制/启动器、上述材料；不复制仓库 docs 目录、研究材料、验收原始数据或任何本地配置。包装启动失败只输出固定分类和包名版本，不输出底层 require/spawn 的本机路径。

内置查询配置及第三方服务限制见 [隐私声明](../../PRIVACY.md)、[免责声明](../../DISCLAIMER.md) 和 [功能限制](../limitations.md)。包内政策文档的公开源码链接指向该次发布的 commit；私有准备产物中的 main 链接只是未验证的占位链接。

参考：[GitHub release 事件](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#release)、[npm provenance](https://docs.npmjs.com/generating-provenance-statements/)、[npm 不可重复发布同名同版本](https://docs.npmjs.com/cli/v11/commands/npm-publish/)。
