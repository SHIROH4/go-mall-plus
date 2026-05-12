# API 接口文档

## 概述

所有 API 通过 API Gateway 统一入口：`http://localhost:8888`

### 统一响应格式

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0=成功，非0=失败 |
| msg | string | 提示信息 |
| data | object | 数据 |

### 认证方式

除登录/注册接口外，其他接口需要在 Header 中携带 Token：

```
Authorization: Bearer <access_token>
```

## 用户服务

### 注册

```
POST /api/user/register
Content-Type: application/json

Request:
{
  "username": "test",
  "password": "password123",
  "email": "test@example.com"
}

Response:
{
  "code": 0,
  "msg": "注册成功",
  "data": {
    "user_id": 1
  }
}
```

### 登录

```
POST /api/user/login
Content-Type: application/json

Request:
{
  "username": "test",
  "password": "password123"
}

Response:
{
  "code": 0,
  "msg": "登录成功",
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 7200
  }
}
```

### 刷新 Token

```
POST /api/user/refresh
Content-Type: application/json

Request:
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}

Response:
{
  "code": 0,
  "msg": "刷新成功",
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

### 获取用户信息

```
GET /api/user/info
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "username": "test",
    "email": "test@example.com",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 退出登录

```
POST /api/user/logout
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "msg": "退出成功"
}
```

---

## 商品服务

### 商品列表

```
GET /api/product/list

Query Parameters:
- page (optional): 页码，默认1
- page_size (optional): 每页数量，默认10
- category_id (optional): 分类ID
- keyword (optional): 搜索关键词

Response:
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "name": "iPhone 15",
        "price": 5999,
        "stock": 100,
        "image": "https://example.com/iphone.jpg",
        "category_id": 1
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 10
  }
}
```

### 商品详情

```
GET /api/product/:id

Response:
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "name": "iPhone 15",
    "desc": "商品描述...",
    "price": 5999,
    "stock": 100,
    "image": "https://example.com/iphone.jpg",
    "category_id": 1,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 获取分类

```
GET /api/product/categories

Response:
{
  "code": 0,
  "msg": "success",
  "data": [
    {"id": 1, "name": "手机"},
    {"id": 2, "name": "电脑"}
  ]
}
```

---

## 购物车服务

### 添加购物车

```
POST /api/cart/add
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "product_id": 1,
  "count": 2
}

Response:
{
  "code": 0,
  "msg": "添加成功"
}
```

### 购物车列表

```
GET /api/cart/list
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "product_id": 1,
      "name": "iPhone 15",
      "price": 5999,
      "count": 2,
      "selected": true
    }
  ]
}
```

### 更新数量

```
PUT /api/cart/update
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "product_id": 1,
  "count": 3
}

Response:
{
  "code": 0,
  "msg": "更新成功"
}
```

### 删除商品

```
DELETE /api/cart/:product_id
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "msg": "删除成功"
}
```

### 清空购物车

```
DELETE /api/cart/clear
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "msg": "清空成功"
}
```

---

## 订单服务

### 创建订单

```
POST /api/order/create
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "product_id": 1,
  "count": 1
}

Response:
{
  "code": 0,
  "msg": "下单成功",
  "data": {
    "order_no": "ORD1234567890",
    "total_amount": 5999,
    "expire_time": 1704067200
  }
}
```

### 订单列表

```
GET /api/order/list
Authorization: Bearer <token>

Query Parameters:
- page (optional): 页码
- page_size (optional): 每页数量
- status (optional): 订单状态 (0=待支付, 1=已支付, 2=已取消, 3=超时)

Response:
{
  "code": 0,
  "msg": "success",
  "data": {
    "list": [
      {
        "id": 1,
        "order_no": "ORD1234567890",
        "product_id": 1,
        "product_name": "iPhone 15",
        "count": 1,
        "total_amount": 5999,
        "status": 0,
        "status_text": "待支付",
        "expire_time": 1704067200,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 10
  }
}
```

### 订单详情

```
GET /api/order/detail
Authorization: Bearer <token>

Query Parameters:
- order_no: 订单号

Response:
{
  "code": 0,
  "msg": "success",
  "data": {
    "id": 1,
    "order_no": "ORD1234567890",
    "product_id": 1,
    "product_name": "iPhone 15",
    "product_desc": "商品描述",
    "count": 1,
    "total_amount": 5999,
    "status": 0,
    "status_text": "待支付",
    "expire_time": 1704067200,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 取消订单

```
POST /api/order/cancel
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "order_no": "ORD1234567890"
}

Response:
{
  "code": 0,
  "msg": "取消成功"
}
```

---

## 支付服务

### 发起支付

```
POST /api/pay/create
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "order_no": "ORD1234567890",
  "pay_type": 1
}

Response:
{
  "code": 0,
  "msg": "支付单创建成功",
  "data": {
    "payment_no": "PAY20240101123456",
    "qr_code": "https://pay.example.com/qr/xxx"
  }
}
```

### 查询支付状态

```
GET /api/pay/status/:payment_no
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "msg": "success",
  "data": {
    "payment_no": "PAY20240101123456",
    "status": 1,
    "status_text": "已支付"
  }
}
```

---

## 地址服务

### 地址列表

```
GET /api/address/list
Authorization: Bearer <token>

Response:
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "user_id": 1,
      "name": "张三",
      "phone": "13800138000",
      "address": "北京市朝阳区xxx",
      "is_default": true
    }
  ]
}
```

### 添加地址

```
POST /api/address/add
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "name": "张三",
  "phone": "13800138000",
  "address": "北京市朝阳区xxx",
  "is_default": true
}

Response:
{
  "code": 0,
  "msg": "添加成功"
}
```

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 签名验证失败 |
| 2001 | 用户不存在 |
| 2002 | 密码错误 |
| 2003 | Token 过期 |
| 2004 | Token 无效 |
| 3001 | 商品不存在 |
| 3002 | 库存不足 |
| 4001 | 订单不存在 |
| 4002 | 订单已取消 |
| 4003 | 订单已超时 |
| 5001 | 支付失败 |
| 5002 | 支付超时 |
