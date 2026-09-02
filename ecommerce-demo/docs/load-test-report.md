# Go-Mall-Plus 下单链路压测与 MySQL 瓶颈定位报告

> 压测时间：2026-09-02 · 环境：本地 Kind 单节点集群（Apple M4）
> 被测链路：`POST /api/order/create`（下单）

---

## 1. 压测环境

| 项 | 配置 |
|---|---|
| 宿主机 | MacBook Air (Mac16,13) · Apple M4 · macOS 26.2 |
| 集群 | Kind 单节点 control-plane · Kubernetes v1.35.0 |
| 服务拓扑 | gateway + 7 个 gRPC 微服务（user/product/order/stock/cart/payment/address） |
| 中间件 | MySQL 8.0（主从，`cpu limit 1000m` / `memory 1Gi`）、Redis Cluster（3 master + 3 replica）、RabbitMQ |
| 压测客户端 | `scripts/order_write_benchmark.go`（本地同机，Go 并发 worker） |

**下单链路**：`gateway → order RPC → product RPC(查商品) + stock RPC(Redis Lua 预扣) → MySQL 单事务(UPDATE stock + INSERT order + INSERT outbox)`，提交后由 Outbox Worker 异步投递 RabbitMQ。

---

## 2. 压测数据

### 2.1 阶梯压测（原始配置 `sync_binlog=1` + `flush=1`）

| 并发 | 请求数 | 成功率 | 吞吐(QPS) | P50 | P90 | P99 |
|---|---|---|---|---|---|---|
| 8 | 100 | 100% | 287 | 20.6ms | 45.9ms | 97.9ms |
| 50 | 2,000 | 100% | 489 | 94.1ms | 132ms | 202ms |
| 100 | 5,000 | 100% | 513 | 176.8ms | 250ms | 288ms |
| 200 | 10,000 | 100% | 516 | 360.7ms | 476ms | 569ms |

**观察**：50 并发即接近吞吐天花板（~500 QPS），之后并发翻倍吞吐几乎不涨、延迟线性上升 → 存在串行瓶颈。

### 2.2 fsync 优化后（`sync_binlog=0` + `flush=2`）

| 并发 | 请求数 | 成功率 | 吞吐(QPS) | P50 | P90 | P99 |
|---|---|---|---|---|---|---|
| 100 | 5,000 | 100% | **708** | 131.7ms | 186ms | 266ms |
| 200 | 8,000 | 100% | **681~748** | 235.7ms | 363ms | 616ms |

**吞吐提升**：100 并发 513→708（**+38%**），200 并发 516→748（**+45%**）。

### 2.3 对照实验

| 场景 | 吞吐(QPS) | 行锁等待增量 |
|---|---|---|
| 固定单商品 900001 | 748 | 4017 次 / 4000 单 |
| 随机 8 商品（`-product-max=8`） | 609 | 3968 次 / 4000 单 |
| READ-COMMITTED 隔离级别 | 749 | 7999 次 / 8000 单 |

---

## 3. 瓶颈定位（对照实验排除法）

| 假设 | 验证方法 | 结论 |
|---|---|---|
| **fsync 强制刷盘** | 运行时调低 `sync_binlog`/`flush` 再压测 | ✅ **确认，+38%** |
| stock 行锁热点 | 随机多商品 vs 单商品，行锁等待 3968 vs 4017（无差异） | ❌ 证伪 |
| 间隙锁 | `REPEATABLE-READ` → `READ-COMMITTED`，吞吐 748 vs 749（无差异） | ❌ 证伪 |
| MySQL CPU 瓶颈 | cgroup `cpu.stat` 采样 ~10% | ❌ 排除 |
| 连接池瓶颈 | `Threads_running=3` / `Threads_connected=20` | ❌ 排除 |

### 关键证据

- **fsync**：`sync_binlog=1` + `innodb_flush_log_at_trx_commit=1` + `innodb_flush_method=fsync`，每个下单事务至少 2 次物理磁盘 fsync。
- **行锁等待 ~1 次/单**：无论单商品/多商品、RR/RC 都稳定存在，是下单事务「UPDATE stock + INSERT order + INSERT outbox」的固有伴随现象，**是果不是因**。

---

## 4. 结论

1. **吞吐天花板 ~750 QPS 由单机 Kind 环境下单链路延迟（~250ms）决定**：下单需经过多次 gRPC 跳转 + Redis Lua + MySQL 三表事务 + Outbox，且所有服务与中间件挤在同一台机器。
2. **唯一确认、且改动成本最低的瓶颈是 fsync 强制刷盘**，优化后吞吐 +38%。
3. **「热点行」并非当前吞吐瓶颈**：stock 行锁、间隙锁均通过对照实验证伪；「分桶 / 雪花 ID / 分表」解决的是更高并发下的次要矛盾。
4. **要真正突破吞吐天花板**，需换部署形态（云服务器 + 独立 MySQL/Redis，消除单机资源争抢），而非单点代码改动。

### 优化建议（按收益/成本排序）

| 优先级 | 方向 | 说明 |
|---|---|---|
| 高 | 独立部署 MySQL/Redis | 消除单机 CPU/IO 争抢，最直接 |
| 中 | fsync 策略权衡 | 生产环境在「数据安全 vs 性能」间权衡（`sync_binlog=0` 有崩溃丢数据风险） |
| 中 | 减少 RPC 跳转 | 合并查商品/预扣的串行调用 |
| 低 | 库存分桶 / 雪花 ID | 高并发（500+）下才显现收益 |

---

## 5. 附：可复现命令

```bash
# 压测工具（支持固定/随机商品）
cd ecommerce-demo
go build -o /tmp/ob scripts/order_write_benchmark.go
/tmp/ob -concurrency=200 -requests=8000 -product-id=900001   # 固定单商品
/tmp/ob -concurrency=200 -requests=8000 -product-max=8        # 随机多商品（对照实验）

# 灌库存（压测前）
STOCK=100000 bash scripts/prepare_order_benchmark_data.sh

# fsync 运行时调低（验证用，压测后恢复）
mysql> SET GLOBAL sync_binlog=0; SET GLOBAL innodb_flush_log_at_trx_commit=2;

# 行锁等待采样
mysql> SHOW GLOBAL STATUS LIKE 'Innodb_row_lock_waits';
```
