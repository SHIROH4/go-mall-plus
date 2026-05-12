# =============================================================================
# Deployment Summary - Kind Kubernetes Cluster (K8s Native)
# =============================================================================

## 部署架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Kind Cluster (ecommerce)                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐                                            │
│  │   Gateway   │  :30088 (NodePort)                         │
│  └─────────────┘                                            │
│       │                                                       │
│  ┌────┴────┬────────┬────────┬────────┬────────┐           │
│  ▼         ▼        ▼        ▼        ▼        ▼            │
│  ┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐     │
│  │  User  │ │Product │ │  Cart  │ │ Order  │ │Payment │     │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘     │
│                                            ┌────────┐        │
│  ┌────────┐                              │ Order  │        │
│  │Address │                               │Workers │        │
│  └────────┘                              └────────┘        │
│                                                              │
│  ┌─────────┐ ┌────────┐ ┌──────────┐                        │
│  │  MySQL  │ │  Redis │ │ RabbitMQ │                        │
│  └─────────┘ └────────┘ └──────────┘                        │
│                                                              │
│  ┌─────────────────┐                                         │
│  │    Frontend     │  :30080 (NodePort)                      │
│  └─────────────────┘                                         │
│                                                              │
│  ─────────────── K8s DNS ───────────────                     │
│  user.ecommerce.svc.cluster.local                             │
│  product.ecommerce.svc.cluster.local                          │
│  order.ecommerce.svc.cluster.local                            │
│  cart.ecommerce.svc.cluster.local                            │
│  payment.ecommerce.svc.cluster.local                          │
│  address.ecommerce.svc.cluster.local                          │
└─────────────────────────────────────────────────────────────┘
```

## 核心特性

- **无 Etcd**: 使用 K8s 原生 DNS 进行服务发现
- **无 Helm**: 直接使用 K8s YAML 部署
- **二进制镜像**: 本地编译 Go 二进制，Docker 运行时镜像
- **配置即代码**: ConfigMap 内嵌服务配置

## 目录结构

```
ecommerce-demo/deploy/kind/
├── kind-config.yaml          # Kind 集群配置
├── namespace.yaml             # K8s 命名空间
├── secrets.yaml               # 密钥配置
├── mysql.yaml                 # MySQL 部署
├── redis.yaml                 # Redis 部署
├── rabbitmq.yaml             # RabbitMQ 部署
│
├── docker/                    # Docker 镜像定义（仅运行时）
│   ├── gateway.dockerfile
│   ├── user.dockerfile
│   ├── product.dockerfile
│   ├── cart.dockerfile
│   ├── order.dockerfile
│   ├── order-delay.dockerfile
│   ├── order-cron.dockerfile
│   ├── order-dlq.dockerfile
│   ├── payment.dockerfile
│   └── address.dockerfile
│
├── services/                  # 服务部署（含 ConfigMap）
│   ├── gateway.yaml
│   ├── user.yaml
│   ├── product.yaml
│   ├── cart.yaml
│   ├── order.yaml             # 包含 order-delay/cron/dlq
│   ├── payment.yaml
│   └── address.yaml
│
├── build.sh                   # 编译脚本
├── deploy-kind.sh             # 部署脚本
├── quick-deploy.sh            # 快速部署脚本
└── README.md

ecommerce-frontend/deploy/kind/
├── docker/
│   ├── Dockerfile             # Nginx 运行时镜像
│   └── nginx.conf             # Nginx 配置
├── frontend.yaml              # Frontend K8s 部署
└── deploy-frontend.sh         # 部署脚本
```

## 快速开始

### 1. 创建 Kind 集群并部署后端

```bash
cd ecommerce-demo/deploy/kind

# 创建集群
./deploy-kind.sh create

# 部署基础设施 (MySQL, Redis, RabbitMQ)
./deploy-kind.sh infra

# 部署应用服务
./deploy-kind.sh services
```

### 2. 编译并加载镜像

```bash
./build.sh full
```

### 3. 部署前端

```bash
cd ../../ecommerce-frontend/deploy/kind
./deploy-frontend.sh full
```

### 4. 一键部署

```bash
cd ecommerce-demo/deploy/kind
./quick-deploy.sh
```

## 访问地址

| 服务 | 地址 |
|------|------|
| Gateway API | http://localhost:30088 |
| Frontend | http://localhost:30080 |
| RabbitMQ | http://localhost:31672 (guest/guest) |

## 服务列表

### 应用服务 (7 个)

| 服务 | 端口 | DNS 名称 |
|------|------|----------|
| gateway | 8888 | gateway.ecommerce.svc.cluster.local |
| user | 8080 | user.ecommerce.svc.cluster.local |
| product | 8081 | product.ecommerce.svc.cluster.local |
| cart | 8083 | cart.ecommerce.svc.cluster.local |
| order | 8082 | order.ecommerce.svc.cluster.local |
| payment | 8084 | payment.ecommerce.svc.cluster.local |
| address | 8085 | address.ecommerce.svc.cluster.local |

### 订单辅助服务 (3 个)

| 服务 | 说明 |
|------|------|
| order-delay | 延迟队列消费者 |
| order-cron | 定时任务 |
| order-dlq | 死信队列消费者 |

### 基础设施 (3 个)

| 服务 | 端口 | 说明 |
|------|------|------|
| mysql | 3306 | MySQL 8.0 |
| redis | 6379 | Redis 7.0 |
| rabbitmq | 5672/15672 | 消息队列 |

## 常用命令

```bash
# 查看状态
./deploy-kind.sh status

# 查看日志
./deploy-kind.sh logs gateway
./deploy-kind.sh logs order

# 清理
./deploy-kind.sh clean
```

## K8s 服务发现原理

K8s 提供内置的服务发现机制：

1. **ClusterIP**: 集群内部访问（默认）
2. **DNS**: 每个 Service 自动获得 DNS 名称
   - 格式: `<service>.<namespace>.svc.cluster.local`
   - 简写: `<service>.<namespace>`
3. **环境变量**: Pod 启动时自动注入同 namespace 的服务地址

本项目使用 DNS 直连，配置示例：

```yaml
UserRpcConf:
  Endpoints:
    - user.ecommerce.svc.cluster.local:8080
```

## 编译说明

本部署方案采用**本地编译二进制 + Docker 运行时镜像**的方式：

1. Go 代码在本地编译为 Linux 二进制
2. 二进制复制到轻量级 Alpine 镜像中
3. 镜像加载到 Kind 集群

这种方式：
- 避免在 Docker 内编译 Go 代码
- 减小镜像体积（< 20MB）
- 加快构建速度
- 提高安全性
