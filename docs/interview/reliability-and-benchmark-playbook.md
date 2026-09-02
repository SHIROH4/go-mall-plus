# Go-Mall-Plus：可靠性加固与压测复盘手册

> 用途：面试复盘、继续压测时的操作手册。本文记录的是本地 Kind 环境中的真实操作与结果；它不能替代生产环境容量证明，也不应直接写成“线上 QPS”。

## 1. 一分钟版本

项目最初已经有微服务、Redis、RabbitMQ 和 Outbox 的设计，但我没有只停留在“实现了”这一层，而是把它拉到本地进行可运行验证：

1. 先让构建、测试和 Kind 部署可复现；
2. 用 Outbox 的租约抢占、CAS 状态更新、Publisher Confirm 和重试/DLQ 处理消息可靠性；
3. 让 MySQL 主从、Redis Cluster、RabbitMQ、Prometheus/Grafana 真正跑起来；
4. 用 RabbitMQ 停机注入验证“先落库、后投递、恢复后补偿”的闭环；
5. 用 `wrk` 对只读接口测出一组零错误基线，并把环境、命令和误差边界记录下来。

面试时可总结为：**我把可靠消息链路从代码设计推进到本地多副本部署、故障恢复和性能基线验证，并且只报告可复现实测数据。**

## 2. 架构与关键链路

### 下单链路

```text
Gateway → Order RPC
            ├─ Redis Lua 预扣库存
            └─ MySQL 同一事务：扣减 DB 库存 + 写订单 + 写两条 Outbox
                                                    │
                                            Outbox Worker（多副本）
                                                    │ Publisher Confirm
                                                    ▼
                                               RabbitMQ
                                      ┌─────────────┴─────────────┐
                                  order.created              延迟超时队列
                                      │                            │
                               幂等消费校验                 到期后取消订单
```

下单同步接口只以“订单、数据库库存、Outbox 是否同事务提交”为成功边界；RabbitMQ 短暂不可用不会让已提交订单丢失。Outbox Worker 稍后补投递。

### Outbox 状态机

| 状态 | 含义 | 转移条件 |
| --- | --- | --- |
| `pending` | 待投递或等待下一次重试 | Worker 原子抢占后进入 `processing` |
| `processing` | 已被某实例取得租约 | Confirm 成功后完成；失败后重试；租约超时可被其他实例接管 |
| `completed` | 已收到 RabbitMQ Publisher Confirm | 终态；清空临时错误和锁 |
| `failed` | 达到最大重试数 | 终态；保留失败原因供人工处理 |

多副本抢占不是“先查后改”。Worker 先挑候选，再以 `id + 状态 + 租约条件` 做条件更新生成 `lock_token`；完成和失败更新也带 `lock_token`。因此慢实例在租约失效后即使继续执行，也不能覆盖接管实例的结果。

## 3. 操作前检查

在仓库根目录执行：

```bash
cd ecommerce-demo
GOCACHE=/tmp/go-mall-plus-gocache go test -p 1 ./...
GOCACHE=/tmp/go-mall-plus-gocache go vet -p 1 ./...
```

说明：`-p 1` 用于降低本地资源竞争，使结果更稳定；项目专用 `GOCACHE` 用于避免沙箱或机器权限影响默认 Go 缓存。

检查部署清单和脚本：

```bash
cd ..
for file in ecommerce-demo/deploy/kind/*.sh ecommerce-demo/scripts/*.sh; do
  bash -n "$file"
done

ruby -e 'require "yaml"; ARGV.each { |f| YAML.load_stream(File.read(f)) }' \
  $(find ecommerce-demo/deploy/kind -name '*.yaml' -type f | sort)

git diff --check
```

面试要点：测试通过只证明已有断言覆盖到的行为；脚本/YAML 校验用于尽早发现语法与缩进问题；真正的消息恢复仍需在运行环境里做故障注入。

## 4. 可复现部署流程

### 4.1 为什么要指定单平台镜像

本机是 Apple Silicon。Kind 节点为 Linux/ARM64。如果 Go 二进制或镜像平台误构建为 AMD64，容器会出现：

```text
exec ./gateway: exec format error
```

构建脚本会通过 Docker 引擎架构确定 `GOARCH`，并显式使用：

```bash
docker buildx build --load --platform linux/arm64 --provenance=false ...
```

`--provenance=false` 的目的是输出单平台镜像，减少 Kind/containerd 对附加 OCI attestation index 的解析差异。部署脚本在本地 `latest` 镜像重新载入后会 `rollout restart`，否则 Kubernetes 因 `imagePullPolicy: Never` 与未变更的 Deployment 不会自动替换旧 Pod。

### 4.2 隔离构建器与集群

当 Docker 构建缓存或旧测试集群状态异常时，使用独立 BuildKit 和独立集群，避免无差别清理其他项目缓存：

```bash
cd ecommerce-demo/deploy/kind
docker buildx create --name go-mall-plus-builder --driver docker-container --bootstrap

KIND_CLUSTER_NAME=ecommerce-clean ./deploy-kind.sh create

GO_MALL_DOCKER_BUILDER=go-mall-plus-builder \
KIND_CLUSTER_NAME=ecommerce-clean \
./build.sh full

KIND_CLUSTER_NAME=ecommerce-clean ./deploy-kind.sh full
```

注意：`kind-config.yaml` 映射了固定宿主机端口 `30088/30300/30909/31672`。若已有另一个 Kind 集群占用它们，需要先停止对应的本地测试集群，不能并行启动两个使用同一端口映射的集群。

### 4.3 基础设施验收

```bash
# Redis Cluster
kubectl exec -n ecommerce redis-0 -- \
  redis-cli -a '<redis-password>' --no-auth-warning cluster info

# MySQL 主从
kubectl exec -n ecommerce mysql-0 -- \
  mysql -uroot -p'<password>' -Nse 'SELECT @@server_id, @@read_only;'
kubectl exec -n ecommerce mysql-1 -- \
  mysql -uroot -p'<password>' -e 'SHOW REPLICA STATUS\G'

# 部署副本状态
kubectl get deployments -n ecommerce
kubectl get pods -n monitoring
```

验收标准：

- Redis：`cluster_state:ok`、`cluster_slots_assigned:16384`、`cluster_known_nodes:6`；
- MySQL：主库 `server_id=1/read_only=0`，从库 `server_id=2/read_only=1`，`Replica_IO_Running` 与 `Replica_SQL_Running` 均为 `Yes`；
- 所有 Deployment 的 `READY` 等于 `DESIRED`，且压测前后重启数为 0。

## 5. 业务与可观测性验收

### 5.1 功能烟雾

```bash
GATEWAY=http://localhost:30088 LOOPS=1 SLEEP=0 PRODUCT_MAX=8 \
  bash ecommerce-demo/scripts/load_test.sh
```

该脚本现在定位为“串行业务烟雾”，不是 QPS 工具：它会注册、登录、浏览商品、加购物车、下单并查看订单。脚本使用 `jq` 解析 JSON，不打印 JWT；商品 ID 限制在现有种子数据的 1–8。

### 5.2 订单与库存一致性检查

```bash
kubectl exec -n ecommerce mysql-0 -- \
  mysql -uroot -p'<password>' ecommerce_demo -e '
  SELECT id, order_no, status, stock_cache_restored FROM `order` ORDER BY id DESC LIMIT 3;
  SELECT id, message_type, status, retry_count FROM outbox ORDER BY id DESC LIMIT 6;
  SELECT product_id, stock_num, version FROM stock WHERE product_id=1;'

kubectl exec -n ecommerce redis-0 -- \
  redis-cli -a '<redis-password>' --no-auth-warning -c GET '{stock}:1'
```

这一步看的是同一笔订单的数据库库存、Redis 库存和 Outbox 是否一致。实测下单后商品 1 的 MySQL 库存由 50 变为 49，Redis `{stock}:1` 也为 49，两条 Outbox 均完成。

### 5.3 Prometheus 与 Grafana

```bash
curl -fsS http://localhost:30909/-/ready
curl -sS http://localhost:30909/api/v1/targets
curl -sS http://localhost:30909/api/v1/rules
curl -fsS http://localhost:30300/api/health
```

本轮验收：Prometheus 4/4 目标为 `up`，加载 4 条告警规则，Grafana 数据库健康。指标包括 Gateway HTTP 延迟/状态码、RPC 调用、下单耗时、MQ 投递/消费以及 Outbox 积压。

一个重要修正：早期 `outbox_pending_count` 错把“本次抢占的数量”当作积压量，批大小是 50，配置 `>1000` 告警永远不可能触发。现在它通过数据库 `COUNT(status=pending)` 记录真实待投递总数；另一个修正是只在 RabbitMQ Confirm 成功后记录 `mq_publish_total`，写入 Outbox 不再被误记作投递成功。

## 6. RabbitMQ 故障注入：完整操作与结论

### 目标

验证 RabbitMQ 暂时不可用时，不会丢订单；恢复后 Outbox 会自动重连、重试并补投递。

### 操作

> 只在隔离测试集群执行，生产环境必须遵循变更和演练流程。

```bash
# 1. 停 RabbitMQ，确认 Pod 已退出
kubectl scale deployment/rabbitmq -n ecommerce --replicas=0
kubectl wait --for=delete pod -l app=rabbitmq -n ecommerce --timeout=60s

# 2. 通过 Gateway 创建订单（使用测试账号 Token）
curl -X POST http://localhost:30088/api/order/create \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <test-token>' \
  -d '{"productId":2,"count":1}'

# 3. 查询对应 orderNo 的 Outbox 状态和 retry_count
kubectl exec -n ecommerce mysql-0 -- \
  mysql -uroot -p'<password>' ecommerce_demo -e '
  SELECT id, message_type, status, retry_count, error_message
  FROM outbox WHERE payload LIKE "%<orderNo>%";'

# 4. 恢复 RabbitMQ，等待 Worker 自动重连
kubectl scale deployment/rabbitmq -n ecommerce --replicas=1
kubectl rollout status deployment/rabbitmq -n ecommerce --timeout=180s

# 5. 再次查询 Outbox
```

### 实测结果

- RabbitMQ 停机期间，下单接口返回成功，订单与库存已落库；
- 两条 Outbox 事件经历了 4 次失败重试；
- RabbitMQ 恢复后，Producer 自动重连，2/2 事件变为 `completed`；
- 完成更新会清空临时 `error_message`，但保留 `retry_count` 作为恢复历史。

### 面试可回答的因果链

“接口成功不是因为 MQ 可用，而是因为订单、库存和 Outbox 在同一个 MySQL 事务提交；MQ 不可用时 Outbox 留在可重试状态。恢复后 Worker 通过租约抢占消息，Publisher Confirm 成功后才 CAS 标记完成。即使 Worker 多副本或有租约过期，也靠 `lock_token` 防止旧实例覆盖新实例状态。”

## 7. 性能基线步骤与解读

完整数据见 [2026-09-02-kind-baseline.md](../benchmarks/2026-09-02-kind-baseline.md)。

### 测试前

1. 确认所有 Pod Ready、重启数为 0；
2. 记录机器、Docker、Kind、Kubernetes、Go 和 Git 版本；
3. 使用只读 `GET /api/product/list`，避免库存不足、订单超时、登录注册等写操作影响结果；
4. 预热或单独记录预热，正式测量期间固定连接数、线程数和时长。

### 命令

```bash
# 稳定零错误基线
wrk -t4 -c16 -d20s --latency http://localhost:30088/api/product/list

# 探索性上限，不作为简历中的稳定容量
wrk -t4 -c32 -d30s --latency http://localhost:30088/api/product/list
```

### 已得到的结果

| 场景 | 吞吐 | P99 | 非 2xx/3xx | 用途 |
| --- | ---: | ---: | ---: | --- |
| 4 线程 / 16 连接 / 20 秒 | 18,461.65 req/s | 20.81 ms | 0 | 可复现的本地稳定读基线 |
| 4 线程 / 32 连接 / 30 秒 | 19,636.37 req/s | 35.70 ms | 125（约 0.021%） | 探索拐点，不能当稳定容量 |

### 为什么不直接把 19.6k QPS 写进简历

这是 Apple M4、Docker Desktop、Kind、本地 8 商品数据集上的单接口读压；它不包含公网、真实数据库规模、写链路、鉴权压力和资源隔离。更严谨的表述是：

> “在 Apple M4 本地 Kind 双副本环境中，对商品列表接口进行 4 线程、16 连接、20 秒压测，得到零非成功响应下约 18.5k req/s、P99 20.81ms 的读基线。”

简历篇幅有限时，应优先报告“完成压测并保留完整口径”，面试时再给上述条件。

## 8. 已执行：受控下单端到端基线

为避免库存耗尽和接口限流混入结论，我新增了独立测试商品准备脚本与固定请求数的下单测试程序：

```bash
bash ecommerce-demo/scripts/prepare_order_benchmark_data.sh
go run ecommerce-demo/scripts/order_write_benchmark.go \
  -requests=30 -concurrency=4 -rate=5
```

它会创建一次性测试账号、调用真实 Gateway 下单接口，并汇总成功率、吞吐、P50/P90/P99 与业务码；不会输出 Token。实测为 30/30 成功，P99 16.51ms，停止后 `outbox_pending_count=0`，数据库中 Outbox 均为 `completed`，所有业务 Pod 保持 Ready、0 重启。

这里的 `-rate=5` 不是“压不动”，而是匹配默认的 `/api/order/create` 入口策略（每 Gateway 副本 5 req/s、burst 10）。未限速提交 20 个请求时恰有 10 个被 429 拦截，证明限流在保护公开写接口。面试应明确：**这验证的是带准入控制的端到端正确性和稳定性，不是订单服务的容量上限。**

随后在隔离集群用 `gateway-benchmark-config.yaml` 临时提高入口配额，100 请求/4 并发为 100% 成功、442.51 req/s、P99 29.92ms；200 请求/8 并发却只有 95% 成功。先排除了 Gateway 限流、go-zero 服务丢弃和 MySQL 死锁，再检查订单号生成方式：两个 Order 副本都固定使用 Snowflake 节点号 `1`，同一毫秒可能生成相同 `order_no`，由数据库唯一键拒绝后被上层包装成“系统拥挤”。

修复为 `ORD + UUID`，并新增 10,000 次唯一性单元测试；同一 200/8 档复测为 100% 成功、479.10 req/s、P99 30.84ms，扩大到 500/8 仍为 100% 成功、544.58 req/s、P99 22.06ms。500/8 结束时短暂积压 700 条 Outbox，约 20 秒后数据库全部为 `completed`、Prometheus `outbox_pending_count=0`、订单号重复查询为 0。另一个独立问题是 MySQL `datetime` 秒精度与事件时间不一致，已通过“生成一次、截断到秒、同值写库和写 Outbox”修复。

面试时可讲成真实闭环：**先把 95% 成功率作为异常而非容量，逐层排除限流、熔断与死锁，定位多副本 Snowflake 节点号重复；修复后用相同档位和更大请求数回归，并验证消息最终排空与订单号唯一性。** 部署中也确认 `build.sh full` 只构建镜像，必须执行 `build.sh load`、滚动重启并等待稳定后才开始压测；否则发布窗口的断连会触发 Gateway 熔断，不能计为业务失败。测试结束后应切回默认 Gateway 限流配置。

## 9. 下一阶段：订单写链路阶梯压测计划

写压测不能复用当前的串行烟雾脚本。推荐按下面顺序执行：

1. 准备固定数量测试用户和库存充足的独立商品；
2. 使用能动态附带 Authorization 的工具脚本，先做 1、4、8、16、32 并发阶梯；
3. 每档至少 60 秒，记录成功/失败数、吞吐、P50/P90/P99、Gateway/Order/Stock 的 CPU/内存、MySQL 连接数、Redis 延迟、Outbox pending、MQ 重试/DLQ；
4. 若失败率超过预设阈值、P99 持续恶化、Outbox 持续积压或任一 Pod 重启，则停止提高并发，保存现场；
5. 压测结束后等待 Outbox 清空，再核对订单数、数据库库存、Redis 库存和队列数量；
6. 对每个异常并发档保留命令、环境和 Grafana 截图，避免只留下一个“最高 QPS”。

建议成功标准（后续可按实际目标修改）：

| 指标 | 初始门槛 |
| --- | --- |
| 请求成功率 | ≥ 99.9% |
| P99 | 先按接口拆分记录，不预设虚假目标 |
| Outbox 积压 | 停压后持续下降并最终归零 |
| 订单/库存一致性 | 所有抽样订单一致，无负库存 |
| Pod 稳定性 | 无 OOM、无异常重启 |

### 已执行的 Redis 写路径阶梯

为了先避免订单库存耗尽干扰压测，已对 `POST /api/cart/add` 执行 4、8、16、32 连接阶梯。32 连接时达到 17,129.53 req/s、P99 37.24ms，且没有非 2xx/3xx；但从 16 到 32 连接的吞吐提升有限、P99 上升，因此停止上探。完整表格在 [性能基线](../benchmarks/2026-09-02-kind-baseline.md)。

面试时要说明这是一条 Redis 写路径，不等同于下单吞吐；它的价值是先确认 JWT、Gateway、Cart RPC 和 Redis Cluster 在并发写入下的稳定边界。

## 10. 高频追问与回答边界

### 为什么既要 Redis Lua，又要数据库库存？

Redis Lua 解决高并发预扣的原子性和缓存侧快速失败；数据库库存与订单、Outbox 同事务提交，是持久化真相。若事务失败，调用库存回滚；超时补偿还会对 Redis 采用幂等脚本，避免重复归还。

### Outbox 能保证“绝对一次”吗？

不能。数据库提交与 MQ Publish 之间仍可能发生“消息已到 MQ、应用尚未来得及标完成”的窗口，因此系统语义是**至少一次投递 + 消费幂等**。这里的 EventID、消费者幂等和状态 CAS 用来控制重复的影响。

### Publisher Confirm 解决什么？

它确认 Broker 已接收发布。没有 Confirm，网络中断时调用方可能不知道消息是否真正到达 Broker；Confirm 成功后才将 Outbox 标为完成。

### 为什么 Outbox 要租约和 lock token？

租约避免某个实例崩溃后消息永远卡在 `processing`；lock token 保证旧实例在租约过期后不能把接管实例的消息误标为成功或失败。

### 这组 QPS 能代表生产吗？

不能。它是明确硬件、Docker/Kind、本地数据规模和单只读接口条件下的基线。生产容量必须在近似生产的网络、数据、资源限制和混合读写负载下重新测量。

## 11. 本轮真实结论

- 已完成：代码回归、可复现部署、基础设施验收、核心业务烟雾、RabbitMQ 故障恢复、Prometheus/Grafana 验收、只读性能基线；
- 已完成：购物车 Redis 写路径阶梯、受 Gateway 限流保护的下单端到端基线；
- 尚未完成：混合读写压测、以独立压测 Gateway 配额执行的下单高并发阶梯、长时间稳定性、延迟订单到期 30 分钟全链路演练、生产级容量评估；
- 说法原则：只陈述已测事实，明确“本地 Kind 基线”的环境边界。
