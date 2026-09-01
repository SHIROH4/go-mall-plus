# Go-Mall-Plus

基于 go-zero + Kind (K8s) 的微服务电商系统。

## 核心工程链路

```text
创建订单
  ├─ Stock RPC + Redis Lua 原子预扣
  └─ MySQL 事务：扣减库存 + 写入订单 + 写入 Outbox
                       │
                       ▼
             租约抢占式 Outbox Worker
                       │ Publisher Confirm
                       ▼
                    RabbitMQ
             ├─ 有界指数退避重试
             ├─ 消费幂等 + 死信队列
             └─ 延迟订单超时检查
                       │
                       ▼
       MySQL 条件状态迁移 + Redis Lua 幂等库存补偿
```

- **Outbox 多副本安全：** Worker 使用唯一锁令牌和超时租约抢占消息，旧持有者不能覆盖新持有者的处理结果。
- **MQ 可靠性：** 业务事件使用稳定 Event ID，重试和 DLQ 投递经 Publisher Confirm 确认后才 ACK 原消息。
- **超时幂等：** 延迟消息与独立 Cron 可并发触发，只有条件更新的赢家回补 MySQL 库存；Redis 补偿以订单号去重并持久化完成状态。
- **可观测性：** Prometheus 指标覆盖 HTTP/RPC、Outbox 与 MQ 处理结果，Grafana 提供运行时和业务看板。

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端框架 | go-zero (gRPC + REST) |
| 数据库 | MySQL 8.0 (StatefulSet 主从) |
| 缓存 | Redis 7.0 Cluster (6 节点) |
| 消息队列 | RabbitMQ |
| 部署 | Kind (Kubernetes in Docker) |
| 监控 | Prometheus + Grafana |
| 前端 | 静态 HTML (Vue 3 工程化前端待建) |

## 微服务

| 服务 | 端口 | 说明 |
|------|------|------|
| Gateway | 8888 | API 网关，JWT 鉴权，/metrics |
| User | 8080 | 用户注册/登录 |
| Product | 8081 | 商品管理 |
| Order | 8082 | 订单处理 + Outbox Pattern + MQ |
| Cart | 8083 | 购物车 (Redis Cluster) |
| Payment | 8084 | 支付服务 |
| Address | 8085 | 收货地址 |
| Stock | 8086 | 库存服务 (Redis Lua 原子扣减) |

## 前置要求

- Docker
- Kind (`brew install kind`)
- kubectl

## 代码验证

```bash
cd ecommerce-demo
go test -p 1 ./...
go vet -p 1 ./...
```

GitHub Actions 会对每次 push 和 pull request 执行格式检查、全仓单元测试和静态分析。Proto 生成代码已纳入版本管理，可通过 `ecommerce-demo/scripts/generate_proto.sh` 重新生成。

## 首次部署

```bash
cd ecommerce-demo/deploy/kind

# 完整部署（创建/复用 Kind 集群、构建镜像、初始化/迁移数据库、部署服务）
bash quick-deploy.sh
```

访问地址：
- 前端：http://localhost:3000 （需 `cd ecommerce-demo/frontend && python3 -m http.server 3000`）
- API 网关：http://localhost:30088
- Grafana：http://localhost:30300 (admin/admin)
- RabbitMQ：http://localhost:31672 (guest/guest)

## 日常使用

### 暂停集群（保留所有数据）

```bash
docker stop ecommerce-cluster-control-plane
```

### 恢复集群

```bash
docker start ecommerce-cluster-control-plane

# 修复 Redis Cluster + 刷新服务连接
cd ecommerce-demo/deploy/kind
bash restart.sh
```

### 完整重启

```bash
cd ecommerce-demo/deploy/kind
bash restart.sh
```

脚本会自动：
1. 创建或复用 Kind 集群
2. 构建并加载全部服务镜像
3. 部署 MySQL、RabbitMQ 和 3 主 3 从 Redis Cluster
4. 初始化数据库并执行未应用的 SQL 迁移
5. 启动微服务、订单 Worker、Prometheus 和 Grafana

### 彻底销毁

```bash
cd ecommerce-demo/deploy/kind
./deploy-kind.sh clean
```

## 压测

```bash
cd ecommerce-demo/scripts
GATEWAY=http://localhost:30088 LOOPS=100 bash load_test.sh
```

## 项目结构

```
ecommerce-demo/
├── app/                    # 微服务
│   ├── gateway/            # API 网关
│   ├── user/               # 用户服务
│   ├── product/            # 商品服务
│   ├── order/              # 订单服务 (+ Outbox + MQ Consumer)
│   ├── cart/               # 购物车服务
│   ├── payment/            # 支付服务
│   ├── address/            # 地址服务
│   └── stock/              # 库存服务
├── common/                 # 公共库 (metrics, response, JWT)
├── deploy/
│   ├── kind/               # K8s 部署配置
│   │   ├── services/       # 各服务 Deployment + ConfigMap
│   │   ├── restart.sh      # 完整重启脚本
│   │   ├── quick-deploy.sh # 首次部署脚本
│   │   └── build.sh        # 镜像构建脚本
│   └── sql/                # 数据库初始化 SQL
└── frontend/               # 静态前端
```

## License

MIT
