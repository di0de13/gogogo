# 礼物订单领域设计

版本：v0.1  
日期：2026-09-02  
对应计划：`2.1.2 礼物订单场景`

## 1. 设计目标

本文件把“购买虚拟礼物”从业务描述细化为可实现、可测试、可解释的领域规则。重点不是模拟真实支付，而是展示在下游依赖不可靠、消息可能重复、请求可能重试时，如何保证订单不重复、状态可追踪、失败可恢复。

## 2. 第一版业务假设

- 一个订单只购买一种礼物，可购买多个数量；
- 金额使用整数最小货币单位保存，不使用浮点数；
- 一个订单对应一个观众、一个直播间和一个地区；
- 账户扣币由本地可控的模拟 Wallet Service 完成；
- 礼物发放和营收指标更新通过事件异步完成；
- 不实现真实支付渠道、退款渠道和跨币种结算；
- 订单创建接口采用“可靠受理”语义，客户端通过查询接口获得最终状态。

## 3. 领域对象

### 3.1 Order 订单

记录用户购买意图和最终业务结果，核心字段包括：

- `order_id`：全局唯一订单号；
- `user_id`、`room_id`、`gift_id`、`region`；
- `quantity`：正整数；
- `unit_price`、`total_amount`：整数最小货币单位；
- `currency`：订单创建时固定；
- `status`：订单主状态；
- `payment_status`：模拟扣币状态；
- `fulfillment_status`：礼物发放状态；
- `version`：乐观锁版本；
- `created_at`、`updated_at`。

### 3.2 Idempotency Record 幂等记录

记录客户端请求和订单的绑定关系：

- 唯一键：`(user_id, idempotency_key)`；
- `request_hash`：请求关键字段的摘要；
- `order_id`；
- `created_at`、`expires_at`。

同一用户使用相同幂等键和相同请求摘要时，返回原订单；摘要不同则返回参数冲突，不能覆盖原请求。

### 3.3 Outbox Event 事件

与订单在同一个本地事务中写入，确保订单提交后事件不会静默消失：

- `event_id`：事件全局唯一且不可变；
- `aggregate_type=order`；
- `aggregate_id=order_id`；
- `event_type`；
- `payload`：版本化 JSON；
- `status`：投递状态；
- `attempts`、`next_retry_at`；
- `created_at`、`published_at`。

### 3.4 Consumer Deduplication 消费去重记录

每个具有副作用的消费者记录已经成功处理的 `event_id`，或者使用业务唯一键做幂等约束。不能只依赖 Kafka offset，因为消费者可能在副作用完成前后崩溃。

## 4. 状态模型

### 4.1 订单主状态

```text
CREATED → PROCESSING → SUCCEEDED
                    └→ FAILED
CREATED ────────────→ CANCELED
```

状态含义：

- `CREATED`：订单已写入，但还未开始处理；
- `PROCESSING`：订单已受理，正在等待或执行下游动作；
- `SUCCEEDED`：扣币和礼物发放均已确认，订单完成；
- `FAILED`：经过可重试和补偿后仍不能完成，原因必须可查询；
- `CANCELED`：在未产生不可逆副作用前主动取消。

### 4.2 支付状态

```text
UNPAID → PAYING → PAID
                └→ PAYMENT_FAILED
```

### 4.3 发放状态

```text
PENDING → FULFILLING → FULFILLED
                    └→ FULFILLMENT_FAILED
```

拆分订单、支付和发放状态，是为了避免一个 `status` 同时表达多个下游阶段，导致故障恢复时无法判断应该重试哪一步。

### 4.4 Outbox 状态

```text
NEW → PUBLISHED
 └→ RETRYING → PUBLISHED
             └→ DEAD
```

`DEAD` 不等于数据丢失，而是表示自动投递已经停止，必须由死信处理或人工补偿继续推进。

## 5. 状态迁移规则

| 当前状态 | 触发条件 | 下一状态 | 备注 |
|---|---|---|---|
| CREATED | 订单校验通过并开始处理 | PROCESSING | 只允许一次逻辑开始 |
| PROCESSING | 扣币和发放均成功 | SUCCEEDED | 记录完成时间 |
| PROCESSING | 不可重试业务错误 | FAILED | 保存错误码，不无限重试 |
| CREATED | 用户在处理前取消 | CANCELED | 已扣币则不能直接取消 |
| 任意终态 | 重复请求/重复事件 | 原状态 | 幂等返回，不重复产生副作用 |
| SUCCEEDED/FAILED/CANCELED | 非法迁移请求 | 拒绝 | 记录审计日志 |

状态迁移必须使用条件更新或事务保护，例如：

```sql
UPDATE orders
SET status = 'SUCCEEDED', version = version + 1
WHERE order_id = ? AND status = 'PROCESSING' AND version = ?;
```

## 6. 创建订单接口语义

### 请求关键字段

`user_id`、`room_id`、`gift_id`、`quantity`、`currency`、`region`、`idempotency_key`。

### 响应语义

- 首次请求：创建订单并返回 `order_id`、`status=PROCESSING` 或 `CREATED`；
- 同一幂等键重试：返回原订单及其当前状态；
- 同一幂等键但请求内容不同：返回 `IDEMPOTENCY_CONFLICT`；
- 业务校验失败：不创建订单、不写 Outbox；
- 数据库暂时不可用：返回失败，客户端可使用原幂等键重试；
- 写库成功后接口返回：即使 Kafka 暂时不可用，也不能把订单伪装成创建失败。

## 7. 本地事务边界

创建订单时，在一个 MySQL 本地事务内完成：

1. 校验并占用幂等键；
2. 插入订单；
3. 插入 `ORDER_CREATED` Outbox 事件；
4. 提交事务。

事务提交后由独立投递器发送 Kafka。订单服务不在数据库事务内同步等待 Kafka，也不使用分布式 2PC。

## 8. 事件设计

第一版事件：

| 事件 | 生产时机 | 消费者 | 幂等业务键 |
|---|---|---|---|
| `ORDER_CREATED` | 订单本地事务提交 | 订单处理器 | `order_id` |
| `PAYMENT_CONFIRMED` | 模拟扣币成功 | 订单状态处理器 | `order_id` |
| `GIFT_FULFILLED` | 礼物发放成功 | 营收指标处理器 | `order_id` + `gift_id` |
| `REVENUE_RECORDED` | 营收事实写入 | 看板聚合器 | `event_id` |

事件至少包含：`event_id`、`event_type`、`event_version`、`order_id`、`user_id`、`room_id`、`region`、`occurred_at` 和业务 payload。

## 9. 失败、重试与补偿

| 故障 | 系统行为 | 最终处理 |
|---|---|---|
| 参数或余额不足 | 不重试 | 订单进入 `FAILED`，保留业务错误码 |
| MySQL 提交失败 | 请求失败 | 客户端复用幂等键重试 |
| MySQL 成功、Kafka 失败 | 订单仍视为已受理 | Outbox 投递器重试 |
| Kafka 重复投递 | 消费者检测重复 | 返回成功但不重复扣币/发放 |
| 消费者处理后宕机、offset 未提交 | 再次消费 | 依靠业务幂等安全重放 |
| 下游暂时超时 | 指数退避重试 | 超限进入死信 |
| 死信长期未处理 | 触发告警 | 对账任务或人工补偿 |
| 状态长时间卡在 PROCESSING | 标记异常 | 对账检查下游事实并推进或失败 |

重试只针对可恢复错误；业务拒绝、参数错误和余额不足不能无限重试。

## 10. 对账规则

对账任务按时间窗口扫描：

- 订单存在但没有对应 Outbox：数据完整性异常；
- Outbox 已发布但没有消费结果：检查消费者或重放事件；
- 已扣币但未发放：重试发放或进入人工处理；
- 已发放但营收事实缺失：补写营收事件；
- 订单、扣币和发放金额不一致：冻结自动修复，生成高优先级告警。

对账任务必须可重复运行，修复动作也必须幂等。

## 11. 测试用例清单

- 正常创建订单；
- 相同幂等键重复提交；
- 相同幂等键但请求摘要不同；
- 两个并发请求竞争同一幂等键；
- 订单写入成功但 Outbox 投递延迟；
- Kafka 重复投递；
- 消费者在副作用完成后、提交 offset 前崩溃；
- 可重试错误达到上限进入死信；
- 不可重试业务错误不进入无限重试；
- `PROCESSING` 超时被对账任务发现；
- 已完成订单收到非法状态迁移请求。

## 12. 面试讲法

### 为什么不用“数据库写完直接发 Kafka”

因为数据库提交和 Kafka 发送是两个独立系统，代码顺序不能形成原子性。进程可能在数据库提交后宕机，导致订单存在但事件丢失。Outbox 把事件和订单放在同一个本地事务，之后允许投递器重试。

### 为什么不追求 Exactly Once

端到端 Exactly Once 成本高且边界复杂。这里采用至少一次投递，结合订单唯一键、消费去重表和下游条件更新，把重复消息转化为安全重放问题。

### 为什么拆分多个状态

订单处理包含扣币、发放和营收记录多个阶段。拆分状态后，系统能明确知道失败发生在哪个阶段，补偿任务才能只重试缺失动作，避免重复扣币。

## 13. 下一步输入

下一最低单元为 `2.1.3 增长实验场景`。订单领域已经确定，增长活动需要定义目标人群、A/B 分流、预算和灰度规则，并明确它如何通过订单事件读取营收结果。
