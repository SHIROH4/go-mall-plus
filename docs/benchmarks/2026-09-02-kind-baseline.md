# Kind 本地性能基线（2026-09-02）

> 本文只记录可复现实测结果，不代表生产容量。测试对象为本地 Kind 双副本服务，数据集仅含 8 个商品。

## 环境

- 分支：`feat/reliability-hardening`
- 基准提交：`275febe`，包含当前未提交工作树改动
- 主机：Apple M4，16 GiB，macOS 26.2
- Go：1.25.6 darwin/arm64
- Docker：29.1.3
- Kind：0.31.0，Kubernetes 1.35.0
- 部署：Gateway 2 副本、Product 2 副本、MySQL 1 主 1 从
- 工具：`wrk`
- 接口：`GET /api/product/list`

## 稳定基线

```bash
wrk -t4 -c16 -d20s --latency http://localhost:30088/api/product/list
```

| 指标 | 结果 |
| --- | ---: |
| 请求数 | 369,263 |
| 吞吐 | 18,461.65 req/s |
| 平均延迟 | 1.94 ms |
| P50 | 0.671 ms |
| P90 | 4.75 ms |
| P99 | 20.81 ms |
| 最大延迟 | 58.09 ms |
| 非 2xx/3xx | 0 |

压测后 Gateway、Product、MySQL Pod 均为 0 重启。

## 探索性上限

```bash
wrk -t4 -c32 -d30s --latency http://localhost:30088/api/product/list
```

| 指标 | 结果 |
| --- | ---: |
| 请求数 | 589,196 |
| 吞吐 | 19,636.37 req/s |
| 平均延迟 | 5.15 ms |
| P50 | 1.10 ms |
| P90 | 18.68 ms |
| P99 | 35.70 ms |
| 最大延迟 | 74.20 ms |
| 非 2xx/3xx | 125（约 0.021%） |

这组结果存在少量非成功响应，只用于观察吞吐拐点，不作为零错误容量结论。后续应增加状态码采样、CPU/内存资源曲线，并分别测试登录、购物车和下单写链路。

## 购物车写链路阶梯基线

> 测试接口为 `POST /api/cart/add`。每档会新建测试用户并携带 JWT，因此覆盖 Gateway 鉴权、Cart RPC、Redis Cluster 写入和双副本路由；不创建订单，不消耗商品库存。工具脚本为 `ecommerce-demo/scripts/run_cart_write_benchmark.sh`。

```bash
THREADS=4 CONNECTIONS=<并发连接数> DURATION=20s \
  bash ecommerce-demo/scripts/run_cart_write_benchmark.sh
```

| 连接数 | 时长 | 吞吐 | P50 | P90 | P99 | 非 2xx/3xx |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 4（预热） | 10s | 8,640.39 req/s | 0.342ms | 1.00ms | 1.73ms | 0 |
| 8 | 20s | 13,381.34 req/s | 0.475ms | 1.16ms | 4.03ms | 0 |
| 16 | 20s | 16,162.99 req/s | 0.762ms | 9.51ms | 31.22ms | 0 |
| 32 | 20s | 17,129.53 req/s | 1.29ms | 17.65ms | 37.24ms | 0 |

压测后 Gateway、Cart、Redis Cluster Pod 均为 Ready、0 重启。32 连接时吞吐增幅明显小于尾延迟增幅，因此本轮没有继续上探；当前可将 8 连接档作为更稳健的本地写路径参考，16/32 连接用于观察饱和趋势。

Prometheus 在 5 分钟窗口内观察到 `/api/cart/add` 均为 200，Outbox 积压为 0。该本地 Kind 环境未安装 Metrics Server，`kubectl top pods` 返回 `Metrics API not available`，故本轮没有记录 CPU/内存数值；后续安装 Metrics Server 或通过 cAdvisor/Prometheus 补齐资源曲线后，再做长时间压测。

## 订单写链路：受 Gateway 限流保护的基线

> 接口为 `POST /api/order/create`，覆盖 JWT 鉴权、Gateway、Order/Stock RPC、Redis Lua 预扣、MySQL 事务与 Outbox 异步投递。为避免把接口保护策略误判成服务错误，本轮以生产样式的默认路由阈值 `5 req/s、burst 10` 进行受控发送；这不是订单服务的并发容量测试。

```bash
# 准备独立压测商品；只写入本地 Kind 测试数据
bash ecommerce-demo/scripts/prepare_order_benchmark_data.sh

# 速率受控下单；脚本新建测试账号，不输出 JWT
go run ecommerce-demo/scripts/order_write_benchmark.go \
  -requests=30 -concurrency=4 -rate=5
```

| 指标 | 结果 |
| --- | ---: |
| 请求数 | 30 |
| 成功/失败 | 30 / 0 |
| 成功率 | 100% |
| 实际吞吐 | 5.16 req/s |
| P50 / P90 / P99 | 10.87 / 15.80 / 16.51 ms |
| 最大延迟 | 16.65 ms |
| API 业务码 | 30 个 `0` |
| 压测后 Outbox | 88 个 `completed`，`pending` 指标为 0 |

压测后全部业务 Pod 均为 Ready、重启数为 0。此前以未限速方式提交 20 个订单时，前 10 个请求成功、后 10 个返回业务码 `429`（“操作太频繁”）；这与路由的 burst=10 完全一致，说明公开下单入口的限流策略生效，不能记为订单服务失败或用来推导写链路容量。

若要进行订单服务的阶梯容量测试，应使用单独的**压测环境 Gateway 配置**提高入口配额，同时仍保留生产默认的限流配置，并逐档核对库存、订单、Outbox 和 MQ；不得直接修改或放宽生产配置。

## 订单写链路：压测专用入口（短批次）

> 使用 `ecommerce-demo/deploy/kind/services/gateway-benchmark-config.yaml` 临时切换 Gateway 的下单入口配额；测试结束后已切回默认 `5 req/s、burst 10` 配置并删除集群中的临时 ConfigMap。该文件只可用于隔离 Kind 环境。

| 请求数 / 并发 | 成功率 | 成功吞吐 | P50 | P90 | P99 | 结论 |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| 100 / 4 | 100% | 442.51 req/s | 7.78ms | 10.82ms | 29.92ms | 首轮短批次通过 |
| 200 / 8（修复前） | 95% | 439.47 req/s | 16.78ms | 20.06ms | 32.49ms | 多副本订单号碰撞，停止上探 |
| 200 / 8（修复后） | 100% | 479.10 req/s | 14.51ms | 21.79ms | 30.84ms | 原失败档回归通过 |
| 500 / 8（修复后） | 100% | 544.58 req/s | 13.83ms | 18.08ms | 22.06ms | 订单号重复 0；Outbox 最终排空 |

8 并发档的失败不是 Gateway 限流或 MySQL 死锁：两个 Order Deployment 副本都固定使用 Snowflake 节点号 `1`，同一毫秒内会生成相同 `order_no`，触发数据库唯一键冲突后被上层泛化为“系统拥挤”。修复为 `ORD + UUID`，保留数据库唯一约束作最终兜底，并新增 10,000 次订单号唯一性单元测试。修复后在同一原失败档 200/8 和扩大到 500/8 的测试均为 100% 成功；500/8 停压后 Outbox 从 700 条 `pending` 在约 20 秒内排空，Prometheus 的 `outbox_pending_count` 回到 0，数据库查询无重复订单号。

压测还发现 MySQL `datetime` 是秒精度，而 Outbox 事件携带未归一化时间可能触发延迟事件秒级校验不一致；已将过期时间在写库和写 Outbox 前统一截断到秒。部署验证时也发现 `build.sh full` 仅构建镜像、不自动加载 Kind；标准流程必须执行 `build.sh load` 后再滚动重启并等待 Pod 和 Gateway 熔断窗口稳定，避免把发布窗口的短暂 RPC 断连误记为应用性能失败。
