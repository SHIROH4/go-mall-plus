# 接口文档

本文档详细描述电商商城后端微服务的所有接口。

## 一、项目概述

### 1.1 微服务架构
```
┌─────────────────────────────────────────────────────────────┐
│                        客户端 (Frontend)                    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway (:8888)                       │
│  - JWT 鉴权                                                  │
│  - 路由转发                                                  │
│  - 参数校验                                                  │
└─────────────────────────────────────────────────────────────┘
        │              │              │              │
        ▼              ▼              ▼              ▼
┌──────────┐  ┌──────────────┐  ┌──────────┐  ┌──────────────┐
│  User    │  │   Product    │  │   Cart   │  │    Order     │
│  Service │  │   Service    │  │  Service │  │   Service    │
└──────────┘  └──────────────┘  └──────────┘  └──────────────┘
        │              │              │              │
        ▼              ▼              ▼              ▼
┌──────────────────────────────────────────────────────────────┐
│                        MySQL Database                        │
└──────────────────────────────────────────────────────────────┘
```

### 1.2 技术栈
- **网关**: Go-Zero REST API
- **服务通信**: gRPC
- **数据库**: MySQL
- **缓存**: Redis
- **服务发现**: Etcd
- **消息队列**: RabbitMQ (订单延迟队列)

### 1.3 鉴权机制
- **算法**: RS256 (RSA 非对称加密)
- **AccessToken**: 有效期 2 小时
- **RefreshToken**: 有效期 7 天
- **黑名单**: Redis 存储失效 Token
- **Header**: `Authorization: Bearer <token>`

---

## 二、用户服务接口

### 2.1 用户注册
**请求**
```http
POST /api/user/register
Content-Type: application/json

{
  "username": "testuser",
  "password": "123456"
}
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1
  }
}
```

**字段说明**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名，4-20字符 |
| password | string | 是 | 密码，最少6字符 |

### 2.2 用户登录
**请求**
```http
POST /api/user/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "123456"
}
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "accessToken": "eyJhbGciOiJSUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJSUzI1NiIs..."
  }
}
```

**字段说明**
| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 | 用户ID |
| accessToken | string | 短期访问令牌 |
| refreshToken | string | 刷新令牌 |

### 2.3 刷新 Token
**请求**
```http
POST /api/user/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGciOiJSUzI1NiIs..."
}
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "accessToken": "eyJhbGciOiJSUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJSUzI1NiIs..."
  }
}
```

### 2.4 获取用户信息
**请求**
```http
GET /api/user/info
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "username": "testuser"
  }
}
```

### 2.5 退出登录
**请求**
```http
POST /api/user/logout
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success"
}
```

---

## 三、商品服务接口

### 3.1 获取商品列表
**请求**
```http
GET /api/product/list
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "products": [
      {
        "id": 1,
        "name": "iPhone 15 Pro",
        "desc": "苹果旗舰手机",
        "price": 799900,
        "image_url": "https://example.com/iphone.jpg",
        "category_id": 1,
        "category_name": "数码电子",
        "stock": 100
      }
    ]
  }
}
```

### 3.2 获取商品详情
**请求**
```http
GET /api/product/:id
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "product": {
      "id": 1,
      "name": "iPhone 15 Pro",
      "desc": "苹果旗舰手机",
      "price": 799900,
      "image_url": "https://example.com/iphone.jpg",
      "category_id": 1,
      "category_name": "数码电子",
      "stock": 100
    }
  }
}
```

**字段说明**
| 字段 | 类型 | 说明 |
|------|------|------|
| price | int64 | 价格，单位为分 |
| stock | int32 | 库存数量 |

### 3.3 分页获取商品
**请求**
```http
GET /api/product/list/page?category_id=1&keyword=iPhone&page=1&page_size=10
```

**参数说明**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category_id | int64 | 否 | 分类ID |
| keyword | string | 否 | 搜索关键词 |
| page | int32 | 否 | 页码，默认1 |
| page_size | int32 | 否 | 每页数量，默认10 |

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "products": [...],
    "total": 50,
    "page": 1,
    "page_size": 10
  }
}
```

### 3.4 获取商品分类
**请求**
```http
GET /api/product/categories
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "categories": [
      {
        "id": 1,
        "name": "数码电子",
        "icon": "digital",
        "sort": 1
      }
    ]
  }
}
```

---

## 四、购物车服务接口

> ⚠️ 所有接口需要登录认证

### 4.1 添加到购物车
**请求**
```http
POST /api/cart/add
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "product_id": 1,
  "product_name": "iPhone 15 Pro",
  "price": 799900,
  "image_url": "https://example.com/iphone.jpg",
  "count": 1
}
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "添加成功",
    "total_count": 5
  }
}
```

### 4.2 获取购物车列表
**请求**
```http
GET /api/cart/list
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "items": [
      {
        "product_id": 1,
        "product_name": "iPhone 15 Pro",
        "price": 799900,
        "image_url": "https://example.com/iphone.jpg",
        "count": 2,
        "selected": true,
        "created_at": 1704067200,
        "updated_at": 1704067200
      }
    ],
    "total_count": 2,
    "total_amount": 1599800
  }
}
```

### 4.3 更新购物车数量
**请求**
```http
PUT /api/cart/update
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "product_id": 1,
  "count": 3
}
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "更新成功"
  }
}
```

### 4.4 删除购物车商品
**请求**
```http
DELETE /api/cart/:product_id
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "删除成功"
  }
}
```

### 4.5 清空购物车
**请求**
```http
DELETE /api/cart/clear
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "清空成功",
    "removed_count": 5
  }
}
```

### 4.6 选择/取消商品
**请求**
```http
PUT /api/cart/select
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "product_id": 1,
  "selected": true
}
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "操作成功"
  }
}
```

### 4.7 获取已选商品
**请求**
```http
GET /api/cart/selected
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "items": [...],
    "selected_count": 3,
    "total_amount": 2399700
  }
}
```

---

## 五、订单服务接口

> ⚠️ 所有接口需要登录认证

### 5.1 创建订单（普通下单/秒杀）
**请求**
```http
POST /api/order/create
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "productId": 1,
  "count": 1
}
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "orderNo": "ORD20240101120000001"
  }
}
```

**说明**
- 普通下单和秒杀使用同一接口
- 后端会自动检查库存
- 库存不足返回错误

### 5.2 获取订单列表
**请求**
```http
GET /api/order/list?page=1&page_size=10&status=0
Authorization: Bearer <accessToken>
```

**参数说明**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int32 | 否 | 页码，默认1 |
| page_size | int32 | 否 | 每页数量，默认10 |
| status | int32 | 否 | 订单状态，0=待支付 1=已支付 2=已取消 3=已超时 |

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "orders": [
      {
        "id": 1,
        "order_no": "ORD20240101120000001",
        "product_id": 1,
        "product_name": "iPhone 15 Pro",
        "count": 1,
        "total_amount": 799900,
        "status": 0,
        "status_text": "待支付",
        "create_time": 1704067200,
        "pay_time": 0
      }
    ],
    "total": 5,
    "page": 1,
    "page_size": 10
  }
}
```

**订单状态说明**
| 状态值 | 状态文本 | 说明 |
|--------|----------|------|
| 0 | 待支付 | 订单创建，等待用户支付 |
| 1 | 已支付 | 用户已完成支付 |
| 2 | 已取消 | 用户主动取消 |
| 3 | 已超时 | 超过支付时限自动取消 |

### 5.3 获取订单详情
**请求**
```http
GET /api/order/detail?order_no=ORD20240101120000001
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "order_no": "ORD20240101120000001",
    "product_id": 1,
    "product_name": "iPhone 15 Pro",
    "product_desc": "苹果旗舰手机",
    "product_image": "https://example.com/iphone.jpg",
    "count": 1,
    "total_amount": 799900,
    "status": 0,
    "status_text": "待支付",
    "create_time": 1704067200,
    "pay_time": 0,
    "expire_time": 1704070800
  }
}
```

### 5.4 取消订单
**请求**
```http
POST /api/order/cancel
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "order_no": "ORD20240101120000001"
}
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "success": true,
    "message": "订单已取消"
  }
}
```

---

## 六、支付服务接口

> ⚠️ 所有接口需要登录认证

### 6.1 发起支付
**请求**
```http
POST /api/pay/create
Authorization: Bearer <accessToken>
Content-Type: application/json

{
  "order_no": "ORD20240101120000001",
  "amount": 799900,
  "pay_channel": "alipay"
}
```

**参数说明**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| order_no | string | 是 | 订单号 |
| amount | int64 | 是 | 支付金额（分） |
| pay_channel | string | 是 | 支付渠道，alipay 或 wechat |

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "payment_no": "PAY20240101120000001",
    "qr_code": "https://qr.alipay.com/xxx",
    "expire_time": 1704070800
  }
}
```

### 6.2 查询支付状态
**请求**
```http
GET /api/pay/status/:payment_no
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "payment_no": "PAY20240101120000001",
    "status": 0,
    "status_text": "待支付",
    "amount": 799900,
    "order_no": "ORD20240101120000001"
  }
}
```

**支付状态说明**
| 状态值 | 状态文本 | 说明 |
|--------|----------|------|
| 0 | 待支付 | 支付单创建，等待用户支付 |
| 1 | 已支付 | 用户已完成支付 |
| 2 | 已取消 | 用户取消支付 |
| 3 | 已超时 | 超过支付时限自动关闭 |

### 6.3 取消支付
**请求**
```http
POST /api/pay/cancel/:payment_no
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "支付已取消"
  }
}
```

### 6.4 支付记录列表
**请求**
```http
GET /api/pay/list?page=1&page_size=10
Authorization: Bearer <accessToken>
```

**响应**
```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "payments": [
      {
        "id": 1,
        "payment_no": "PAY20240101120000001",
        "order_no": "ORD20240101120000001",
        "amount": 799900,
        "status": 1,
        "status_text": "已支付",
        "pay_channel": "alipay",
        "pay_time": 1704068000,
        "expire_time": 1704070800,
        "created_at": 1704067200
      }
    ],
    "total": 10,
    "page": 1,
    "page_size": 10
  }
}
```

---

## 七、错误码说明

### 通用错误码
| code | 说明 |
|------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（Token无效或过期） |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

### 业务错误码
| code | 错误信息 | 说明 |
|------|----------|------|
| 1001 | 用户名已存在 | 注册时用户名重复 |
| 1002 | 用户名或密码错误 | 登录失败 |
| 2001 | 商品不存在 | 商品ID错误 |
| 2002 | 库存不足 | 库存不够 |
| 3001 | 购物车为空 | 结算时无商品 |
| 4001 | 订单不存在 | 订单号错误 |
| 4002 | 订单已取消 | 重复取消 |
| 4003 | 订单已支付 | 重复支付 |
| 4004 | 订单已超时 | 支付超时 |
| 5001 | 支付单不存在 | 支付单号错误 |
| 5002 | 支付已取消 | 重复取消 |
| 5003 | 支付已超时 | 支付超时 |

---

## 八、业务流程说明

### 8.1 用户登录流程
```
1. 用户输入用户名密码
2. 调用 /api/user/login
3. 后端验证用户密码
4. 后端签发 AccessToken 和 RefreshToken
5. 前端保存 Token 到 localStorage
6. 请求时携带 AccessToken
```

### 8.2 Token 刷新流程
```
1. 请求时收到 401 响应
2. 检查是否有 RefreshToken
3. 调用 /api/user/refresh 刷新 Token
4. 刷新成功后重试原请求
5. 刷新失败则跳转登录页
```

### 8.3 普通下单流程
```
1. 用户浏览商品列表
2. 进入商品详情页
3. 点击"加入购物车"
4. 进入购物车页面
5. 选择商品，点击"去结算"
6. 进入确认订单页面
7. 点击"提交订单"
8. 调用 /api/order/create
9. 订单创建成功，跳转支付页
```

### 8.4 秒杀下单流程
```
1. 进入秒杀专区
2. 选择秒杀商品
3. 点击"立即抢购"
4. 前端防抖检查
5. 直接调用 /api/order/create
6. 后端 Redis 扣减库存
7. 订单创建成功
8. 跳转支付页
```

### 8.5 支付流程
```
1. 用户选择支付方式
2. 点击"确认支付"
3. 调用 /api/pay/create 创建支付单
4. 获取支付二维码
5. 轮询调用 /api/pay/status 查询支付状态
6. 支付成功后跳转订单页
```
