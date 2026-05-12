# Go-Mall-Plus

基于 go-zero + Vue3 的全栈微服务电商系统，支持 Docker Compose 和 Kubernetes 部署。

## 项目概览

| 前端 | 后端 |
|------|------|
| Vue 3 + Vite | go-zero 微服务 |
| Element Plus UI | gRPC + REST |
| TypeScript | MySQL + Redis + RabbitMQ |

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端框架 | Vue 3 + TypeScript + Vite |
| UI 组件 | Element Plus |
| 状态管理 | Pinia |
| 后端框架 | go-zero |
| 服务通信 | gRPC (内部) + REST (网关) |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis |
| 消息队列 | RabbitMQ |
| 部署 | Docker Compose / Kubernetes |

## 快速开始

### 前置要求

- Go 1.21+
- Node.js 16+
- Docker & Docker Compose

### 克隆项目

```bash
git clone https://github.com/your-username/go-mall-plus.git
cd go-mall-plus
```

### 一键启动 (推荐)

```bash
# 启动所有服务（后端 + 前端 + 基础设施）
docker-compose up -d

# 查看状态
docker-compose ps
```

访问地址：
- 前端：http://localhost:3000
- API 网关：http://localhost:8888
- RabbitMQ：http://localhost:15672 (guest/guest)

### 手动启动

**后端：**
```bash
cd ecommerce-demo
docker-compose up -d
```

**前端：**
```bash
cd ecommerce-frontend
npm install
npm run dev
```

## 项目结构

```
go-mall-plus/
├── docker-compose.yml      # 统一编排配置
├── Makefile               # 构建脚本
├── docs/                  # 文档
│   ├── architecture.md   # 架构设计文档
│   └── api.md           # API 接口文档
├── ecommerce-demo/        # 后端微服务
│   ├── app/             # 各微服务
│   │   ├── gateway/     # API 网关
│   │   ├── user/        # 用户服务
│   │   ├── product/     # 商品服务
│   │   ├── order/       # 订单服务
│   │   ├── cart/        # 购物车服务
│   │   ├── payment/     # 支付服务
│   │   └── address/     # 地址服务
│   ├── common/          # 公共库
│   └── deploy/          # 部署配置
│       ├── docker/      # Docker 构建
│       └── kind/        # Kubernetes
└── ecommerce-frontend/   # 前端应用
    ├── src/
    │   ├── api/         # 接口封装
    │   ├── views/       # 页面组件
    │   └── stores/      # 状态管理
    └── deploy/          # 前端部署
```

## 系统架构

```
┌──────────────────────────────────────────────────────────────┐
│                         Client                               │
│                     (Vue 3 Browser)                         │
└────────────────────────────┬─────────────────────────────────┘
                             │ HTTP/REST
                             ▼
┌──────────────────────────────────────────────────────────────┐
│                     API Gateway (:8888)                      │
│                   (go-zero REST + JWT)                       │
└───────┬────────────┬────────────┬────────────┬──────────────┘
        │            │            │            │
        │ gRPC       │ gRPC       │ gRPC       │ gRPC
        ▼            ▼            ▼            ▼
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│   User   │  │  Product  │  │  Order   │  │   Cart   │
│  :8080   │  │  :8081   │  │  :8082   │  │  :8083   │
└────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │             │             │
     ▼             ▼             ▼             ▼
┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐
│  MySQL  │   │  Redis  │   │RabbitMQ │   │  Redis  │
│         │   │  Cache  │   │   MQ    │   │  Cart   │
└─────────┘   └─────────┘   └────┬────┘   └─────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    ▼             ▼             ▼
              ┌──────────┐ ┌──────────┐ ┌──────────┐
              │  Delay   │ │   Cron   │ │   DLQ    │
              │ Consumer │ │  Scanner │ │Consumer  │
              └──────────┘ └──────────┘ └──────────┘
```

## 功能模块

### 后端微服务

| 服务 | 端口 | 说明 |
|------|------|------|
| Gateway | 8888 | API 网关，JWT 鉴权 |
| User | 8080 | 用户注册/登录 |
| Product | 8081 | 商品管理 |
| Order | 8082 | 订单处理 + MQ 异步 |
| Cart | 8083 | 购物车 (Redis) |
| Payment | 8084 | 支付服务 |
| Address | 8085 | 收货地址 |

### 前端页面

- 登录/注册
- 首页（轮播、分类、秒杀）
- 商品列表/详情
- 购物车
- 确认订单
- 订单列表/详情
- 支付页面
- 个人中心

## 详细文档

- [架构设计](./docs/architecture.md) - 系统设计、技术选型
- [接口文档](./docs/api.md) - API 接口详细说明

## 配置说明

### 生成 JWT 密钥对

```bash
cd ecommerce-demo/deploy/cert
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem
```

### 环境变量

修改 `ecommerce-demo/deploy/configs/` 下的配置文件：

```yaml
# 数据库
MYSQL_HOST: localhost
MYSQL_PORT: 3306
MYSQL_USER: root
MYSQL_PASSWORD: your_password

# Redis
REDIS_HOST: localhost
REDIS_PORT: 6379

# RabbitMQ
RABBITMQ_HOST: localhost
RABBITMQ_PORT: 5672
```

## 开发指南

### 后端开发

```bash
cd ecommerce-demo

# 编译所有服务
make build

# 运行单个服务
go run app/gateway/gateway.go -f app/gateway/etc/gateway.yaml
```

### 前端开发

```bash
cd ecommerce-frontend

# 安装依赖
npm install

# 开发模式
npm run dev

# 构建生产版本
npm run build
```

## Kubernetes 部署

```bash
cd ecommerce-demo/deploy/kind

# 创建集群
kind create cluster --name ecommerce

# 部署
kubectl apply -f namespace.yaml
kubectl apply -f infrastructure/
kubectl apply -f services/
```

## License

MIT License
