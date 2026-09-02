# 关键状态机与迁移契约

版本：v0.1  
日期：2026-09-02  
对应计划：`2.2.2 关键状态机`

## 1. 设计原则

- 状态表示已经确认的业务事实，不表示某个函数执行到哪一步；
- 每次迁移必须有明确触发事件、前置状态和后置状态；
- 终态不可被普通请求覆盖；
- 重复事件返回幂等成功，不重复产生副作用；
- 非法迁移被拒绝并留下审计信息；
- 当前状态保存在主表，详细原因和历史写入追加到状态历史或事件日志。

## 2. 订单状态机

### 2.1 主状态

```text
CREATED ──start──► PROCESSING ──success──► SUCCEEDED
   │                    │
 cancel                 └─business_error──► FAILED
   ▼
CANCELED
```

允许迁移：

| 当前 | 事件 | 下一状态 | 执行者 |
|---|---|---|---|
| CREATED | `ORDER_ACCEPTED` | PROCESSING | Order Application |
| CREATED | `ORDER_CANCELED` | CANCELED | Order Application |
| PROCESSING | `ORDER_COMPLETED` | SUCCEEDED | Order Consumer |
| PROCESSING | `ORDER_FAILED` | FAILED | Order Consumer/Compensator |

禁止迁移：

- `SUCCEEDED` → 任意状态；
- `FAILED` → `PROCESSING`，除非补偿任务创建显式修复记录；
- `CANCELED` → `PROCESSING`；
- 任何状态直接跳到 `SUCCEEDED` 而没有支付和发放事实。

### 2.2 支付状态

```text
UNPAID ──begin──► PAYING ──confirmed──► PAID
                      │
                      └─rejected──► PAYMENT_FAILED
```

`PAID` 是不可重复扣币的事实。重复 `PAYMENT_CONFIRMED` 事件必须返回幂等成功，不得再次写入扣币账本。

### 2.3 发放状态

```text
PENDING ──begin──► FULFILLING ──done──► FULFILLED
                       │
                       └─failed──► FULFILLMENT_FAILED
```

发放使用业务唯一键 `(order_id, gift_id)`。重试只能推进未完成的发放记录，不能重新扣币。

## 3. Outbox 状态机

```text
NEW ──publish──► PUBLISHED
 │
 └─temporary_error──► RETRYING ──publish──► PUBLISHED
                            │
                            └─max_attempts──► DEAD
```

规则：

- `NEW` 事件可被多个投递器竞争，但只能有一个成功更新发布状态；
- `PUBLISHED` 不能回到 `NEW`；需要重放时创建显式 replay 任务；
- `DEAD` 代表自动重试停止，不代表删除 payload；
- 投递状态和消费者业务状态独立，Kafka 成功不等于下游业务成功。

## 4. 活动状态机

```text
DRAFT ──submit──► PENDING_REVIEW ──approve──► APPROVED
  │                      │                       │
  └─edit───────────────┘                         └─gray──► GRAYING
                                                        │
                                                        ├─promote──► ACTIVE ──time_end──► EXPIRED
                                                        ├─pause────► PAUSED ──resume──► ACTIVE
                                                        └─rollback─► ROLLED_BACK
```

规则：

- `DRAFT` 只能编辑，不能被资格判断命中；
- `PENDING_REVIEW` 不能修改核心规则，需撤回后生成新草稿；
- `APPROVED` 才能进入灰度；
- `GRAYING` 和 `ACTIVE` 可以暂停；
- `ROLLED_BACK` 和 `EXPIRED` 是业务终态，不能恢复原版本；
- 回滚控制新分流，不修改已创建订单。

## 5. 状态迁移接口

领域层只暴露受约束的迁移方法，不允许业务代码任意设置字符串状态：

```go
type OrderState string

const (
	OrderCreated    OrderState = "CREATED"
	OrderProcessing OrderState = "PROCESSING"
	OrderSucceeded  OrderState = "SUCCEEDED"
	OrderFailed     OrderState = "FAILED"
	OrderCanceled   OrderState = "CANCELED"
)

type TransitionError struct {
	EntityID string
	From     string
	Event    string
	Reason   string
}

func (o *Order) Apply(event OrderEvent) error {
	// 只允许状态机表中定义的迁移，并校验事件前置条件。
	return nil
}
```

持久化层再使用版本号或条件更新防止并发覆盖；领域状态机解决“能不能迁移”，数据库条件更新解决“并发时谁先成功”。

## 6. 幂等和并发规则

### 6.1 请求幂等

- 先用 `(user_id, idempotency_key)` 唯一约束抢占请求；
- 已存在且 `request_hash` 相同：返回原订单；
- 已存在但摘要不同：返回冲突；
- 并发竞争由数据库唯一约束裁决，应用层处理 duplicate key 并读取原记录。

### 6.2 事件幂等

- 消费前检查 `event_id` 或业务唯一键；
- 副作用和去重记录尽量放在同一事务中；
- 去重记录已存在时，提交 offset 但不重复执行副作用；
- 消费者在副作用事务提交前崩溃时，消息再次投递仍然安全。

### 6.3 版本并发控制

订单和活动更新使用 `version` 乐观锁：

```sql
UPDATE campaign_versions
SET status = ?, version = version + 1
WHERE campaign_id = ? AND version = ? AND status = ?;
```

影响行数为 0 时，调用方重新读取状态并返回冲突，而不是覆盖其他操作者的变更。

## 7. 状态历史和审计

所有人工控制动作和异常迁移记录：

- `entity_type`、`entity_id`；
- `from_state`、`event`、`to_state`；
- `operator_id` 或系统任务 ID；
- `reason`、`trace_id`；
- `created_at`。

状态历史是追加写，不允许更新或删除。当前状态可以从主表快速读取，审计和复盘依赖历史记录。

## 8. 状态机测试矩阵

- 每个允许迁移至少一个成功测试；
- 每个禁止迁移至少一个拒绝测试；
- 重复事件不会改变状态或产生重复副作用；
- 并发迁移只有一个请求成功，另一个得到版本冲突；
- 终态收到任何普通迁移请求都保持不变；
- Outbox 超过重试次数进入 `DEAD`；
- 活动回滚后新用户不再命中，历史订单状态不变；
- 从数据库恢复的状态不能绕过领域校验直接写成终态。

## 9. 面试叙事

状态机不是为了把字符串封装得更复杂，而是为了把非法业务动作变成显式错误。订单状态解决业务事实，版本号解决并发覆盖，Outbox 状态解决投递生命周期，审计历史解决“为什么变成这个状态”的追溯问题。三者职责不同，不能用一个 `status` 字段或一个数据库锁全部替代。

## 10. 下一步输入

第 2 阶段已完成。下一步进入 `2.3.1 30 秒、2 分钟、5 分钟项目讲解`，将业务和架构文档压缩成面试可用的不同长度叙事。
