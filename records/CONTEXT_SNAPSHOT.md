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

工作区初始化、`2.1.1 直播间和用户场景`、`2.1.2 礼物订单场景`、`2.1.3 增长实验场景`、`2.2.1 Room、Order、Growth 服务边界`、`2.2.2 关键状态机`、`2.3.1 30 秒、2 分钟、5 分钟项目讲解`、`2.3.2 一页架构图和订单时序图`、`3.1.1 配置、日志、错误和优雅退出` 已完成，Go 平台包测试通过。业务基线见 `docs/01-BUSINESS-REQUIREMENTS.md`，订单领域见 `docs/02-ORDER-DOMAIN.md`，增长实验见 `docs/03-GROWTH-EXPERIMENT.md`，服务边界见 `docs/04-SERVICE-BOUNDARIES.md`，状态机见 `docs/05-STATE-MACHINES.md`，面试讲解见 `docs/06-INTERVIEW-PITCH.md`，架构图见 `docs/07-ARCHITECTURE-DIAGRAMS.md`。由于沙箱限制，运行 Go 命令时使用 `GOCACHE=/private/tmp/livegrow-gocache`；本地 TCP 监听验证需在本机或 CI 补跑。下一步是 `3.1.2 HTTP API、请求校验和统一响应`。

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
- 第 2 阶段已完成复盘，下一阶段从文档转向可运行 Go 代码。

## 状态机摘要

- 订单主状态：`CREATED → PROCESSING → SUCCEEDED/FAILED`，处理前可取消；
- 支付和发放状态独立维护，终态不可被普通请求覆盖；
- Outbox：`NEW → PUBLISHED`，临时失败进入 `RETRYING`，超限进入 `DEAD`；
- 活动：`DRAFT → PENDING_REVIEW → APPROVED → GRAYING → ACTIVE`，可暂停、回滚或过期；
- 领域状态机判断迁移是否合法，数据库版本号处理并发覆盖，审计历史记录原因；
- 第 2 阶段复盘记录见 `records/reviews/PHASE-02-DOMAIN-MODELING.md`；
- 第 2 阶段复盘已完成；当前优先将设计转为可运行 Go 代码与测试。

## 面试讲解摘要

- 30 秒：直播营收增长平台 + Go/MySQL/Redis/Kafka + Outbox/幂等 + AI 活动护栏；
- 2 分钟：讲清 Room/Order/Growth、订单可靠受理、异步处理和稳定分流；
- 5 分钟：展开服务边界、offer 快照、预算并发、故障隔离和 AI Code Review；
- 必须区分已实现、已设计、待实现；不虚构生产规模或性能；
- 高风险问题已有诚实回答模板；架构图和订单时序图已完成；
- 第 2.3 阶段复盘记录见 `records/reviews/PHASE-2.3-INTERVIEW-MATERIALS.md`；
- 下一步从讲解材料转向 Go 工程实现。

## 工程底座摘要

- 配置：环境变量 + 默认值 + duration/日志级别校验；
- 日志：标准库 `log/slog` JSON 结构化输出，支持 source；
- 错误：客户端安全消息与内部 cause 分离；
- 生命周期：HTTP server 统一封装，支持 SIGINT/SIGTERM 和超时 Shutdown；
- 健康检查：`/healthz`；
- 测试：平台包单测通过；当前沙箱禁止 TCP 监听，HTTP 集成验证待在本机或 CI 执行；
- 下一步实现统一 HTTP 响应、请求 ID 和参数校验。
