# 可迁移上下文快照

更新时间：2026-09-02

## 项目目标

30 天内准备字节跳动后端开发实习生（营收增长方向）面试，使用真实可运行的 Go 项目证明后端工程、分布式系统和 AI 工程能力。

## 候选人主线

北航计算机大三（28 届），AI 论文、AI 初创公司实习、算法竞赛获奖。当前主线是：AI/算法背景 + 主动补齐 Go 后端 + 直播营收分布式场景。

## 项目定位

`LiveGrow`：直播间、虚拟礼物订单、营收事件、增长实验和 AI 运营 Copilot。

## 技术主线

Go、MySQL、Redis、Kafka；Outbox、本地事务、幂等消费、重试/死信、缓存治理、限流、可观测性、多 region 故障模拟。

## 关键真实性边界

不声称真实亿级生产规模；性能数据必须注明本地环境；未实现能力用设计或模拟描述；AI 不直接操作账务和权限。

## 文件入口

- 总计划：`docs/PLAN.md`
- 状态：`docs/STATUS.md`
- AI 过程日志：`records/AI_VIBE_LOG.md`
- Go 项目：`project/livegrow/`

## 当前状态

工作区初始化、`2.1.1 直播间和用户场景`、`2.1.2 礼物订单场景`、`2.1.3 增长实验场景`、`2.2.1 Room、Order、Growth 服务边界` 已完成，Go 最小程序测试通过。业务基线见 `docs/01-BUSINESS-REQUIREMENTS.md`，订单领域见 `docs/02-ORDER-DOMAIN.md`，增长实验见 `docs/03-GROWTH-EXPERIMENT.md`，服务边界见 `docs/04-SERVICE-BOUNDARIES.md`。由于沙箱限制，运行 Go 命令时使用 `GOCACHE=/private/tmp/livegrow-gocache`。下一步是 `2.2.2 关键状态机`。

## 业务基线摘要

- 角色：Viewer、Creator、Operator、Risk/Reviewer、Platform；
- 主链路：进房 → 查看活动 → 购买礼物 → 订单可靠受理 → Outbox/Kafka → 异步发放和营收统计；
- 强约束：订单唯一性、模拟扣币幂等、营收事实可追溯；
- 最终一致：礼物发放、实时营收看板、热度和在线人数；
- AI 边界：只生成候选活动配置，必须经过 Schema、预算、权限、审批和灰度；
- 非目标：真实支付、真实全球部署、真实用户数据、复杂推荐和完整前端后台。

## 订单领域摘要

- 订单主状态：`CREATED → PROCESSING → SUCCEEDED/FAILED`，处理前可取消；
- 支付和礼物发放各自维护阶段状态，避免状态语义混杂；
- `(user_id, idempotency_key)` 唯一，重复请求返回原订单，请求摘要不同则冲突；
- 订单与 `ORDER_CREATED` Outbox 在同一个 MySQL 本地事务内提交；
- Kafka 采用至少一次投递，消费者用 `event_id`/业务唯一键保证副作用幂等；
- 可恢复错误重试，超限进入死信；对账任务发现卡单、缺事件和金额不一致；
- 订单可靠性主线独立于增长实验，增长只通过事件和 offer 快照接入。

## 增长实验摘要

- 活动生命周期：`DRAFT → PENDING_REVIEW → APPROVED → GRAYING → ACTIVE`，可暂停、回滚或过期；
- 同一活动版本使用 `hash(campaign_id + user_id)` 稳定分流；
- 活动总预算和单用户预算均使用整数金额和原子条件更新；
- 第一版优先 MySQL 作为预算真相，Redis 不承担预算最终一致性；
- 营收指标消费可靠事件异步聚合，消费者必须幂等且可重放；
- 回滚只停止未来新分流，不修改历史订单和奖励；
- AI 生成 JSON 活动草稿，经过 Schema、权限、地区、时间、金额、模拟评估和人工审批后才能灰度。

## 服务边界摘要

- Room 拥有直播间元信息和热点读模型；Order 拥有订单、幂等、Outbox、模拟钱包账本和发放记录；Growth 拥有活动、版本、分组、预算和指标投影；
- 第一版采用模块化单体，逻辑边界先于部署边界；
- 服务不直接读写对方的表；Growth 消费订单事件建立自己的指标投影；
- Order 不同步调用 Growth，使用带版本和签名的 offer 快照；
- Kafka 事件统一包含 `event_id`、类型/版本、生产者、region、聚合 ID、时间和 trace ID；
- 非核心服务故障不能拖垮订单事实链路；
- 下一步统一订单、活动和 Outbox 状态机。
