# 本地开发与质量门禁

版本：v0.1  
日期：2026-09-05  
对应计划：`3.1.4 工程命令和本地开发配置`

## 1. 统一命令

在 `project/livegrow` 目录执行：

| 命令 | 作用 |
|---|---|
| `make fmt` | 格式化所有 Go 文件 |
| `make fmt-check` | 检查是否有未格式化文件 |
| `make test` | 运行全部测试 |
| `make vet` | 运行 Go 静态分析 |
| `make coverage` | 生成并打印覆盖率报告 |
| `make check` | 格式检查 + 测试 + vet |
| `make run` | 启动服务 |

Makefile 默认将 `GOCACHE` 指向 `/private/tmp/livegrow-gocache`，解决当前沙箱默认 Go 缓存目录不可写的问题。开发者可以通过 `make GOCACHE=/path/to/cache test` 覆盖。

## 2. 配置约定

`.env.example` 只记录变量名和安全示例值，不保存密钥。当前配置由环境变量读取：

- `LIVEGROW_ENV`：运行环境；
- `LIVEGROW_HTTP_ADDR`：HTTP 监听地址；
- `LIVEGROW_SHUTDOWN_TIMEOUT`：优雅退出超时；
- `LIVEGROW_LOG_LEVEL`：`debug/info/warn/error`。

引入新配置时，必须同时补充默认值、校验、`.env.example` 和配置单测。

## 3. AI 生成代码的质量门禁

每次 AI 生成或修改 Go 代码后，至少执行：

```text
make fmt
make check
```

如果涉及并发、状态或存储，再增加：

- `go test -race ./...`；
- 相关状态机和幂等测试；
- 集成测试或故障演练；
- 代码审查记录。

`make check` 通过不代表业务正确，只能说明格式、基础测试和静态分析通过。

## 4. 本地环境与 CI 差异

- 当前沙箱禁止 TCP 监听，因此 `make run` 的真实端口验证需在本机或 CI 执行；
- 本地或 CI 应补充 `/healthz` HTTP 集成测试；
- 真实 MySQL、Redis、Kafka 接入前，先使用容器或测试替身验证，再记录环境和版本。

## 5. 面试可讲点

我把 AI 代码生成纳入统一质量门禁：先格式化，再测试和静态分析；涉及并发或状态变化时增加 race、幂等和故障测试。工具通过只代表工程基本面通过，不能替代业务正确性判断。

## 6. 下一步

进入订单状态机领域代码实现，沿用 `make check` 作为每个最低单元的基础验收。
