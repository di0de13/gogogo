# Room、Order、Growth 服务边界

版本：v0.1  
日期：2026-09-02  
对应计划：`2.2.1 Room、Order、Growth 服务边界`

## 1. 设计目标

将业务需求拆成三个清晰的领域边界，同时控制原型复杂度：

- 每类核心数据只有一个服务负责写入；
- 服务之间通过 API 或事件通信，不直接读写对方的表；
- 订单主链路不依赖增长看板和热度数据；
- 第一版可以用模块化单体运行，边界稳定后再拆成独立进程。

“逻辑边界先于部署边界”是本项目的重要工程取舍：不为了展示微服务而过早引入网络、注册中心和分布式部署复杂度。

## 2. 服务职责和数据所有权

| 服务 | 负责什么 | 自有数据 | 明确不负责 |
|---|---|---|---|
| Room Service | 直播间元信息、直播状态、热点读模型 | `rooms`、`room_stats`、直播间缓存 | 订单、活动预算、账务事实 |
| Order Service | 礼物订单、模拟扣币、礼物发放编排、可靠事件 | `orders`、`idempotency_records`、`outbox_events`、`wallet_ledger`、`fulfillment_records` | 活动配置、实验分组、看板聚合 |
| Growth Service | 活动版本、资格判断、实验分组、预算和指标投影 | `campaigns`、`campaign_versions`、`assignments`、`campaign_budget`、`metric_daily` | 修改订单状态、直接扣币、发放礼物 |

### 2.1 共享基础设施不等于共享数据所有权

- MySQL 可以在原型中使用同一个实例，但按服务划分表和迁移目录；
- Redis 可以使用同一个实例，但 key 必须带服务和 region 前缀；
- Kafka 是公共消息基础设施，Topic 的生产者和消费者责任必须明确；
- 任何服务不得因为“同库方便”直接查询另一个服务的表。

## 3. 调用和依赖关系

```text
                         ┌──────────────┐
                         │ API Gateway  │
                         └──┬────┬────┬──┘
                            │    │    │
                       read │    │    │ create/query
                            ▼    ▼    ▼
                         Room Growth Order
                           │      ▲      │
                           │      │      │ publish events
                           └──────┴──────┴──────► Kafka
                                                  │
                                           Growth Metrics Consumer
```

推荐依赖方向：

- Gateway 可以编排三个服务；
- Room 不依赖 Order 或 Growth；
- Order 不读取 Growth 的数据库，不依赖 Growth 看板；
- Growth 消费 Order 事件，不反向修改 Order；
- Order 只依赖自己的 Wallet Adapter 和消息基础设施。

## 4. 同步 API 契约

### 4.1 Room Service

| 方法 | 接口 | 说明 |
|---|---|---|
| GET | `/v1/rooms/{room_id}` | 返回直播间详情、状态、热点读数和可用礼物摘要 |
| GET | `/v1/rooms/{room_id}/eligibility-context` | 返回用于活动判断的公开房间标签，不返回订单数据 |
| POST | `/internal/rooms/{room_id}/stats` | 内部更新热度读模型，允许异步和丢弃旧值 |

### 4.2 Growth Service

| 方法 | 接口 | 说明 |
|---|---|---|
| GET | `/v1/growth/offers?user_id=&room_id=&region=` | 返回当前用户可见的活动和不可变 offer 快照 |
| POST | `/v1/campaigns/drafts` | 创建运营或 AI 活动草稿 |
| POST | `/v1/campaigns/{id}/submit-review` | 提交审核 |
| POST | `/v1/campaigns/{id}/gray` | 设置灰度比例 |
| POST | `/v1/campaigns/{id}/pause` | 暂停新分流 |
| POST | `/v1/campaigns/{id}/rollback` | 回滚当前版本 |
| GET | `/v1/campaigns/{id}/metrics` | 查询最终一致的实验指标 |

### 4.3 Order Service

| 方法 | 接口 | 说明 |
|---|---|---|
| POST | `/v1/orders` | 创建礼物订单，支持幂等键 |
| GET | `/v1/orders/{order_id}` | 查询订单、支付、发放和事件处理状态 |
| POST | `/internal/orders/{order_id}/reconcile` | 内部对账或补偿入口，必须审计 |

订单请求可携带 Growth 返回的不可变 `offer_snapshot`，包括 `campaign_id`、`campaign_version`、`variant`、奖励规则版本和签名。Order 只验证快照格式、签名和时间，不在事务中调用 Growth，从而避免活动服务故障阻塞订单。

## 5. 数据读取规则

### 5.1 允许的读取

- Gateway 调用服务公开 API；
- Growth 消费 Order 事件建立自己的指标投影；
- Order 使用请求中的 offer 快照，不读取 Growth 表；
- Room 提供公开房间上下文，Growth 根据上下文判断资格。

### 5.2 禁止的读取

- Order 直接查询 `campaigns` 或 `assignments`；
- Growth 直接查询 `orders` 统计 GMV；
- Room 直接查询用户订单判断主播营收；
- 任一服务绕过领域服务直接更新别人的表。

这样做的代价是数据存在短暂延迟或需要事件投影，但换来了领域自治和故障隔离。

## 6. 跨服务事件契约

### 6.1 Order 发布

| Topic | 事件 | 主要消费者 |
|---|---|---|
| `livegrow.order.events` | `ORDER_CREATED` | Order Processor |
| `livegrow.order.events` | `PAYMENT_CONFIRMED` | Order、Growth |
| `livegrow.order.events` | `GIFT_FULFILLED` | Growth Metrics、主播营收读模型 |
| `livegrow.revenue.events` | `REVENUE_RECORDED` | Growth Metrics、对账任务 |

### 6.2 Growth 发布

| Topic | 事件 | 主要消费者 |
|---|---|---|
| `livegrow.growth.events` | `CAMPAIGN_PUBLISHED` | Gateway 缓存刷新、审计 |
| `livegrow.growth.events` | `CAMPAIGN_PAUSED` | Gateway 缓存刷新、告警 |
| `livegrow.growth.events` | `CAMPAIGN_ROLLED_BACK` | Gateway 缓存刷新、审计 |

### 6.3 公共事件信封

所有跨服务事件使用统一信封：

```json
{
  "event_id": "evt_01...",
  "event_type": "GIFT_FULFILLED",
  "event_version": 1,
  "producer": "order-service",
  "region": "sg",
  "aggregate_id": "order_01...",
  "occurred_at": "2026-09-02T10:00:00Z",
  "trace_id": "trace_01...",
  "payload": {}
}
```

事件消费者必须忽略未知字段、校验版本和必填字段，并将不可解析事件送入隔离队列而不是无限重试。

## 7. Offer 快照和跨服务解耦

Growth 返回的是“可验证的优惠快照”，不是让 Order 在创建订单时实时查询活动：

```text
Gateway → Growth：获取 offer
Growth → Gateway：campaign/version/variant/rule/signature
Gateway → Order：携带 offer_snapshot 创建订单
Order：本地验证快照并写订单 + Outbox
```

好处：

- Growth 暂时不可用时，已经拿到快照的订单仍可可靠受理；
- 订单记录了当时的活动版本，后续规则修改不影响历史订单；
- Order 不需要知道 Growth 的内部表结构。

风险和限制：

- 快照可能在获取后过期，因此 Order 需要校验有效期；
- 预算最终扣减仍以订单事实和 Growth 的原子预算规则为准；
- 签名密钥轮换和 offer 重放保护属于后续安全增强项。

## 8. 模块化单体实现映射

第一版代码按领域包组织，即使运行在一个进程中也保持依赖方向：

```text
project/livegrow/
├── cmd/livegrow/
├── internal/
│   ├── room/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   ├── order/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   ├── growth/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapter/
│   └── platform/
│       ├── http/
│       ├── events/
│       └── config/
└── migrations/
```

领域层不依赖 HTTP、MySQL 或 Kafka；adapter 层负责具体实现；application 层编排用例。这样后续拆进程时，主要变化在 adapter 和传输层，而不是重新改业务规则。

## 9. 故障隔离原则

| 故障 | Room | Order | Growth |
|---|---|---|---|
| Redis 不可用 | 降级受保护查询 | 不受影响 | 活动读取降级或返回无活动 |
| Kafka 不可用 | 热点事件可延迟 | 订单写入和 Outbox 继续 | 指标延迟，活动控制保留最近快照 |
| Growth 不可用 | 进房不受影响 | 已有 offer 可继续受理；新 offer 可降级 | 活动读取失败并告警 |
| Order 消费者停止 | 房间和活动读取不受影响 | 订单进入处理中，等待重试 | 指标延迟，可重放 |
| Room 数据库只读 | 使用缓存 | 不受影响 | 使用最近房间上下文或拒绝新分流 |

核心原则：非核心看板或策略读取故障不能拖垮订单事实链路。

## 10. 面试叙事

### 为什么先做模块化单体

我的目标是先验证领域边界和一致性，而不是为了“看起来像微服务”增加部署复杂度。Room、Order、Growth 在代码和数据上已经隔离，后续可以根据流量和团队边界拆成独立服务。

### 为什么 Growth 不直接查 Order 表

直接查表虽然简单，但会让两个服务共享数据模型和索引演进，最终形成隐式耦合。我让 Growth 消费订单事件建立自己的指标投影，接受看板的最终一致，换取订单服务的自治和故障隔离。

### 为什么 Order 不同步调用 Growth

订单主链路不应依赖活动看板或策略服务。通过带签名的 offer 快照传递活动版本，Order 可以在本地验证并可靠落库；Growth 短暂故障不会造成订单事实丢失。

## 11. 下一步输入

下一最低单元为 `2.2.2 关键状态机`，将把订单、活动和 Outbox 的状态迁移规则整理成统一的状态机文档和 Go 可测试接口。
