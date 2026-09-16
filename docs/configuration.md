# 配置与存储

命令名为 `sushiro-cli`，配置目录仍使用 `sushiro`，路径变量仍为 `SUSHIRO_*`。

## 公共查询

正常 `stores`、`slots` 使用内置上游默认查询配置，无需导入个人凭证。默认值存在于源码和二进制中；运行时不自动写入用户配置，也不读取个人档案。来源、有效期和服务授权边界见 [来源说明](provenance.md)。

```sh
sushiro-cli public status --json
sushiro-cli public import --file /absolute/private/public-import.json
```

`public status` 只检查本地状态，不发请求，也不输出秘密。`configuration_source=builtin_default`、`persisted=false` 表示内存默认；此时 `configured=false` 仅表示未导入有效的非空覆盖配置。`server_validity=unknown` 不代表在线验证成功。

仅当你有自行取得且有权使用的查询配置时才需要覆盖。导入文件为 UTF-8 JSON，`schema_version` 为整数 `1`、`profile` 为选定档案名、`base_url` 固定为 `https://crm-cn-prd.sushiro.com.cn`；其余字段均为字符串：`query_authorization`、`x_app_code`、`x_app_client`、`user_agent`、`referer`。不要将值放进命令行、环境变量、日志或 issue。请在本地私有编辑器中准备文件，使用受限目录和文件权限。

导入拒绝未知、重复、大小写变体、null 字段及追加 JSON；上限 64 KiB。公共配置不能含 `wechat_id`、`phone_number`、`reservation_authorization`，即使值为空。

| 情况 | 行为 |
| --- | --- |
| 无显式配置文件 | 使用内置公共默认值 |
| 显式文件损坏、权限不安全或最终文件为符号链接 | `public_config_invalid`，不回退 |
| 显式查询值为空 | 查询报 `public_config_required`，不回退 |
| 远端公共请求返回 401/403 | `public_query_rejected`，不要求个人微信登录 |

## 个人会话

个人功能仅支持导入本人授权取得的已有业务会话，不提供原生登录或通用会话获取向导。

```sh
sushiro-cli auth import --file /absolute/private/credentials.json
sushiro-cli auth status --json
```

个人 JSON 使用相同 `schema_version`、`profile`、`base_url` 和客户端头字段，另支持字符串 `reservation_authorization`、`wechat_id`、`phone_number`、`query_authorization`。公共查询不会借用个人文件中的查询值。导入校验和 `auth status.complete` 只表示本地字段状态，不验证服务端有效性，也不推断过期时间。功能限制见 [支持范围](limitations.md)。

## 文件位置与清理

| 系统 | 默认个人根目录 |
| --- | --- |
| macOS | `~/Library/Application Support/sushiro` |
| Linux | `$XDG_CONFIG_HOME/sushiro`，未设置时 `~/.config/sushiro` |
| Windows | `%APPDATA%\sushiro`；当前安全文件存储不支持 |

个人档案为根目录的 `<profile>.json`，公共覆盖为 `public/<profile>.json`。`SUSHIRO_CONFIG_DIR` 可覆盖个人根目录；`SUSHIRO_PUBLIC_CONFIG_DIR` 可独立覆盖公共目录。变量仅传专用绝对路径，公共与个人存储目录必须不同。

文件是未加密 JSON；macOS/Linux 使用目录 0700、文件 0600，不自动修正已有不安全权限。Windows 导入和持久化操作不受支持，无文件的内置公共默认模式不依赖安全存储；这不代表 Windows 已完成原生验收。

升级或卸载不自动删除导入配置。本版没有清理命令；需要清理时停止 CLI/MCP，核实实际路径和 profile，再手动删除对应 JSON，并处理导入副本及备份。删除公共覆盖会恢复内置默认；删除个人文件不撤销服务端会话或取消预约，也不保证安全擦除。详见 [隐私说明](../PRIVACY.md)。
