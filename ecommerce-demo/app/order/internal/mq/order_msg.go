package mq

import (
    "time"
)

/*
  延迟订单消息结构
  用于发送到延迟队列，30分钟后触发超时检查

  设计考量：
  1. 消息体尽可能轻量，减少MQ存储压力
  2. 只传必要字段，订单详情通过OrderNo从DB查询
  3. 创建时间用于日志追踪
*/
type DelayOrderMsg struct {
    OrderNo    string    `json:"orderNo"`    // 订单号
    ProductID  int64     `json:"productId"`  // 商品ID（用于库存回滚）
    Count      int32     `json:"count"`      // 购买数量（用于库存回滚）
    CreateTime time.Time `json:"createTime"` // 消息创建时间（用于日志）
    ExpireTime time.Time `json:"expireTime"` // 订单过期时间（校验用）
}

// BuildDelayOrderMsg 构建延迟订单消息
func BuildDelayOrderMsg(orderNo string, productID int64, count int32, expireTime time.Time) *DelayOrderMsg {
    return &DelayOrderMsg{
        OrderNo:    orderNo,
        ProductID:  productID,
        Count:      count,
        CreateTime: time.Now(),
        ExpireTime: expireTime,
    }
}
