# 使用与输出

以下示例假定已将从源码构建的 `sushiro-cli` 放在 PATH。安装来源和当前状态见 [README](../README.md)。

```sh
sushiro-cli stores --near 39.97,116.43 --limit 2 --json
sushiro-cli stores --store 3006 --json
sushiro-cli slots --store 3006 --adult 2 --child 0 --table T --json
sushiro-cli public status --json
sushiro-cli auth status --json
sushiro-cli interactive
```

`--near LAT,LON` 为参照点；`--limit N` 范围是 1–10000。默认坐标 `1,1` 不表示用户位置。门店输出的 `wait` 为等位桌数；时段的 `availability` 是查询时点状态，不能视作名额保证。桌型沿用 `T`/`C` 编码。

`--profile NAME` 默认为 `default`，允许 `[A-Za-z0-9][A-Za-z0-9_-]{0,63}`。配置与个人功能见 [配置说明](configuration.md)。

## 实验性个人功能

`ticket-status` 查询当前排队票和预约票；`reservations` 尝试查询完整预约列表。它们需要个人会话，当前成功路径尚未完成验收；旧列表接口已知返回 404。不存在数据与查询失败必须区分。

`reserve` 需要门店、日期、时间、成人数和桌型，可指定儿童数；`cancel` 需要票据 ID。具体参数以 `sushiro-cli help` 为准。日期为 `YYYYMMDD`、时间为 `HHMMSS`，均指中国大陆门店当地时间。

写操作必须得到用户对具体参数的授权，再提供 `--confirm`；交互模式还要求输入 `yes`。确认参数不是用户授权的替代。当前真实创建、取消及可靠回查均未闭环，不应作为已验证的订位流程。

## 结果与失败处理

`--json` 返回 `ok`、`state`、`source`，以及 `data` 或 `error`。`source=live` 表示服务端结果，`local` 表示本地配置；不会用历史缓存冒充当前预约。个人结果可能含个人信息，分享前需脱敏。

| 退出码 | 含义 |
| --- | --- |
| 0 | 本次操作成功；本地状态成功不代表在线会话有效 |
| 1 | 参数、查询或业务失败 |
| 3 | 写入结果不确定 |

HTTP 请求不自动重试。写入不确定时会尝试一次预约回查；创建仍保持 `uncertain`，取消须观察到同票据明确取消状态才报告成功。回查依赖的旧列表接口尚不可用，因此请通过官方小程序核对，不要自动重提写请求。

公共配置的 `public_config_invalid`、`public_config_required` 与 `public_query_rejected` 见 [配置说明](configuration.md)。
