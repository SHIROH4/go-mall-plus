# Go-Mall-Plus

基于 go-zero + Kind (K8s) 的微服务电商系统。

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

## 首次部署

```bash
cd ecommerce-demo/deploy/kind

# 完整部署（创建 Kind 集群 + 构建镜像 + 部署所有服务）
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
1. 部署 MySQL、RabbitMQ、Redis Cluster
2. 组建 Redis Cluster 并验证
3. 清理旧 MQ 消息
4. 按依赖顺序启动所有微服务
5. 健康检查（商品/购物车/下单）

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
