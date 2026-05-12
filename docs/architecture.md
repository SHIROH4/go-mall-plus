# 系统架构设计

## 整体架构

Go-Mall-Plus 采用前后端分离架构，后端基于 go-zero 框架构建微服务集群。

```
┌─────────────────────────────────────────────────────────────────┐
│                         前端 (Vue 3)                             │
│                   http://localhost:3000                          │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP/REST
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    API Gateway (8888)                           │
│                    go-zero REST + JWT Auth                      │
└───────┬─────────────┬─────────────┬─────────────┬───────────────┘
        │             │             │             │
        │ gRPC        │ gRPC        │ gRPC        │ gRPC
        ▼             ▼             ▼             ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│     User     │ │   Product    │ │    Order     │ │     Cart     │
│    :8080     │ │    :8081    │ │    :8082    │ │    :8083     │
└──────┬───────┘ └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
       │                │                │                │
       ▼                ▼                ▼                ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│    MySQL     │ │    Redis    │ │  RabbitMQ    │ │    Redis     │
│   User DB    │ │   Product   │ │   Order MQ   │ │    Cart      │
│              │ │   Cache     │ │             │ │              │
└──────────────┘ └──────────────┘ └──────┬───────┘ └──────────────┘
                                          │
                    ┌─────────────────────┼─────────────────────┐
                    ▼                     ▼                     ▼
              ┌──────────┐         ┌──────────┐         ┌──────────┐
              │  Order   │         │  Order   │         │  Order   │
              │  Delay   │         │  Cron    │         │   DLQ    │
              │ Consumer │         │ Scanner  │         │ Consumer │
              └──────────┘         └──────────┘         └──────────┘
```

## 技术选型

### 后端

| 组件 | 技术 | 说明 |
|------|------|------|
| 框架 | go-zero | 高性能微服务框架 |
| 网关 | go-zero/rest | 统一入口，JWT 鉴权 |
| RPC | go-zero/zrpc | 高性能 gRPC |
| 数据库 | MySQL 8.0 | 主数据存储 |
| ORM | GORM | Go ORM |
| 缓存 | Redis | 热数据缓存、购物车 |
| 消息队列 | RabbitMQ | 异步订单处理 |
| ID 生成 | Snowflake | 分布式 ID |

### 前端

| 组件 | 技术 | 说明 |
|------|------|------|
| 框架 | Vue 3 | 渐进式框架 |
| 构建 | Vite | 快速开发体验 |
| UI | Element Plus | 企业级 UI |
| 状态 | Pinia | Vue 状态管理 |
| HTTP | Axios | 请求封装 |
| 类型 | TypeScript | 类型安全 |

## 服务详解

### 1. API Gateway

**职责：**
- 接收客户端 HTTP 请求
- JWT Token 验证
- 请求路由到后端 gRPC 服务
- 统一响应格式
- 限流、熔断

**关键技术：**
- go-zero rest 框架
- RSA256 JWT 验证
- 中间件链

### 2. User Service

**职责：**
- 用户注册/登录
- Token 发放（Access + Refresh）
- 用户信息管理

**数据库表：**
- `user` - 用户信息

### 3. Product Service

**职责：**
- 商品 CRUD
- 分类管理
- 库存管理（Redis 缓存）

**数据库表：**
- `product` - 商品信息
- `category` - 分类

**缓存策略：**
- 商品详情 Redis 缓存
- 库存预热到 Redis

### 4. Order Service

**职责：**
- 创建订单
- 订单状态管理
- 超时订单处理

**数据库表：**
- `order` - 订单主表
- `stock` - 库存表

**异步处理：**
- RabbitMQ 接收创建订单请求
- 异步落库，提高响应速度
- 延迟队列处理超时订单

### 5. Cart Service

**职责：**
- 购物车管理
- 全 Redis 存储

**Redis 结构：**
```
cart:{user_id} -> Hash
  product:{product_id} -> {count, price}
```

### 6. Payment Service

**职责：**
- 支付单创建
- 支付状态查询
- 支付回调处理

**数据库表：**
- `payment` - 支付记录

### 7. Address Service

**职责：**
- 收货地址管理

**数据库表：**
- `address` - 收货地址

## 消息流

### 订单创建流程

```
1. Client -> Gateway: POST /api/order/create
2. Gateway -> Order: gRPC CreateOrder
3. Order -> Redis: Lua 原子扣减库存
4. Order -> RabbitMQ: 发送订单消息
5. Order -> Client: 返回订单号
6. (异步) MQ Consumer -> MySQL: 落库
7. (异步) MQ Consumer -> RabbitMQ: 发送延迟消息
8. (异步) Delay Consumer -> 检查超时
```

### 库存扣减流程

```lua
-- Redis Lua 脚本，原子操作
local stock_key = KEYS[1]
local count = tonumber(ARGV[1])

local current = tonumber(redis.call('GET', stock_key) or 0)
if current < count then
    return 0  -- 库存不足
end

redis.call('DECRBY', stock_key, count)
return 1  -- 扣减成功
```

## 数据流

### 秒杀场景

```
1. 商品详情页 -> 预检库存（Redis）
2. 点击购买 -> 前端防抖
3. 网关 -> 限流
4. Order -> Redis Lua 原子扣库存
5. 库存充足 -> MQ 异步落库
6. 库存不足 -> 返回"手慢了"
7. 订单创建成功 -> 跳转支付
```

## 高可用设计

### 服务层面
- 多副本部署
- 健康检查
- 自动重启

### 数据层面
- MySQL 主从
- Redis 持久化
- RabbitMQ 镜像队列

### 网关层面
- 请求限流
- 熔断降级
- 超时控制

## 安全性

### 认证
- JWT RS256 非对称加密
- Access Token (2小时)
- Refresh Token (7天)
- Token 黑名单

### 数据安全
- HTTPS 传输
- 参数校验
- SQL 注入防护
- XSS 防护
