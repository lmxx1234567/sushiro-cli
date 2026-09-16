# MCP

MCP 可选，CLI 可独立使用。macOS/Linux 的 amd64（x64）或 arm64 安装 Node.js 20+ 和 npm 后，可在支持 stdio MCP 的客户端中通过 npx 启动：

```json
{
  "mcpServers": {
    "sushiro-cli": {
      "command": "npx",
      "args": ["--yes", "sushiro-cli@0.1.3", "mcp"]
    }
  }
}
```

这里只指定主包，npm 自动选择平台包，无需手选；Windows 暂不支持此 npx 入口。这里固定包版本，升级时显式修改版本号。首次启动可能下载 npm 包，客户端需能找到 npx 并访问 registry。使用其他档案时，args 为 `["--yes", "sushiro-cli@0.1.3", "--profile", "NAME", "mcp"]`。

也可使用已安装或从源码构建的二进制绝对路径。Windows 请从 [GitHub Release v0.1.3](https://github.com/lmxx1234567/sushiro-cli/releases/tag/v0.1.3) 下载对应架构的独立二进制，使用 `sushiro-cli.exe` 的绝对路径作为 command；JSON 中的 Windows 路径反斜线需写成 `\\`：

```json
{
  "mcpServers": {
    "sushiro-cli": {
      "command": "/absolute/path/sushiro-cli",
      "args": ["mcp"]
    }
  }
}
```

使用其他档案时将 args 改为 `["--profile", "NAME", "mcp"]`。初始化、工具发现不要求个人登录；公共工具使用独立查询配置。

| 工具 | 用途 |
| --- | --- |
| `stores` | 门店列表或单店，支持 `near`、`limit`、`store_id` |
| `slots` | 指定门店和人数的时段 |
| `ticket_status` | 个人当前票据，实验性 |
| `reservations` | 个人预约列表，旧接口已知不可用 |
| `reserve` / `cancel` | 实验性写入，必须有具体用户授权与 `confirm: true` |

公共 `slots` 参数示例：

```json
{"store_id":"3006","adult":2,"child":0,"table_type":"T"}
```

以工具发现返回的 schema 为完整参数契约。stdout 仅承载协议消息，业务错误返回 `isError=true`。MCP 与 CLI 共享 [结果语义](usage.md) 和 [验证限制](limitations.md)；工具存在不代表其业务成功路径已经验证。

MCP 宿主可能保存工具输入、结果和对话，个人结果可能含个人信息；按宿主自身的数据政策评估使用方式，见 [隐私说明](../PRIVACY.md)。
