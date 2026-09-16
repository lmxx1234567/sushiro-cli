# 项目关系、使用边界与责任说明

更新日期：2026-09-16。

本项目是非官方的寿司郎中国服务 CLI/MCP 工具，与寿司郎、其关联企业、微信或腾讯不存在官方隶属、合作或背书关系。相关名称和商标属于各自权利人，仅用于说明兼容对象。

项目参考并改编了 [Ryujoxys/sushiro-overdose](https://github.com/Ryujoxys/sushiro-overdose) 的协议字段、响应处理及公共查询默认配置，感谢其开源工作。来源和许可见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。上游代码公开或采用 MIT，并不意味着第三方服务授权、商标许可或公共 token 分发条件也已获得确认。

## 使用与服务限制

你应仅在有权使用的账户和数据范围内操作，并遵守适用法律及官方服务的现行规则。不要把工具用于冒用账户、绕过访问限制、滥发请求或占用他人服务资源。此使用说明不改变 MIT 对软件本身授予的许可，也不授予第三方服务或账户的访问权。

接口、会话、门店、时段和名额可能随时变化或不可用；查询成功不保证预约成功。等位数表示桌数，不能当作等候分钟数。错误不能解释为“没有预约”，当前票据快照不能替代完整预约历史。

创建和取消预约会影响真实账户。使用前必须明确授权具体门店、日期、时间、人数、桌型或要取消的票据。CLI 的 `--confirm` 和 MCP 的 `confirm=true` 是调用条件，不能证明操作者或模型真的取得了授权；宿主和调用者仍须落实确认。遇到超时、连接中断或 `state=uncertain`，不要盲目重试；先检查返回的回查信息，并通过官方服务核对实际状态。当前完整预约列表不可用时，程序的回查不能保证完成确认。

截至本说明核对的基线，公共默认模式的 CLI/MCP/npm 查询有成功验收记录；个人接口存在 E010/404 的历史失败，原生认证及创建/取消预约闭环仍待验证。历史成功记录不是当前可用性、所有平台兼容性或未来持续服务承诺。

## 不保证与责任限制

本项目新增贡献采用 [MIT 许可证](LICENSE)，第三方内容保留其各自许可，详见 [第三方声明](THIRD_PARTY_NOTICES.md)。依 MIT 的不保证与责任限制安排，在适用法律允许的最大范围内，软件按现状提供，不对适销性、特定用途适用性或不侵权作明示或默示保证；作者和版权持有人的相关责任受该许可证限制。本文不扩张许可证所授予的权利，也不对第三方服务作保证。[MIT 官方文本](https://opensource.org/license/mit)

**任何不保证或责任限制都不排除、限制适用法律不允许排除或限制的责任，也不剥夺依法不可放弃的权利。** 本说明不是对“所有风险均由用户承担”或“维护者在任何情形下均免责”的承诺。具体效力取决于实际发布主体、使用场景、适用法律及事实。

本文是项目说明，未经律师审定，不是面向特定法域的法律意见或合规认证。发布账号/署名已确定为 `lmxx1234567`，但法律主体身份、管辖/适用法域、私密联系渠道及第三方接口使用条件尚待确认；仅面向中文用户不构成选择适用法律的依据。功能边界见[已知限制](docs/limitations.md)，数据处理方式见 [PRIVACY.md](PRIVACY.md)。

## English summary

This is an unofficial project with no affiliation or endorsement from Sushiro, WeChat or Tencent. Credit and source attribution belong to Ryujoxys/sushiro-overdose and the other named dependencies. Interface availability and reservation success are not guaranteed. Account writes require explicit authorization; uncertain writes must not be blindly retried. New project contributions use MIT; third-party terms remain applicable. Warranty and liability limitations apply only to the extent permitted by applicable law and do not exclude non-excludable liability or rights. This document is not legal advice or a compliance certification.
