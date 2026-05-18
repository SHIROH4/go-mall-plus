package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"ecommerce-demo/app/order/internal/config"
	"ecommerce-demo/app/order/internal/mq"
	"ecommerce-demo/app/order/internal/repo"
	"ecommerce-demo/app/order/pb"
	"ecommerce-demo/common/metrics"
	productclient "ecommerce-demo/app/product/product"
	stockclient "ecommerce-demo/app/stock/stock"

	"github.com/bwmarrin/snowflake"
)

var (
	ErrStockNotEnough = errors.New("手慢了，商品库存不足")
	ErrProductInvalid = errors.New("无效的商品")
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *pb.CreateOrderReq) (*pb.CreateOrderResp, error)
	ConfirmPay(ctx context.Context, req *pb.ConfirmPayReq) (*pb.ConfirmPayResp, error)
	ListOrder(ctx context.Context, req *pb.ListOrderReq) (*pb.ListOrderResp, error)
	GetOrderDetail(ctx context.Context, req *pb.GetOrderDetailReq) (*pb.GetOrderDetailResp, error)
	CancelOrder(ctx context.Context, req *pb.CancelOrderReq) (*pb.CancelOrderResp, error)
}

type orderServiceImpl struct {
	repo          repo.OrderRepo
	outboxRepo    repo.OutboxRepo
	productRpc    productclient.Product
	stockRpc      stockclient.Stock
	producer      mq.Producer
	snowflakeNode *snowflake.Node
	timeoutConfig config.OrderTimeoutConfig
}

func NewOrderService(
	repo repo.OrderRepo,
	outboxRepo repo.OutboxRepo,
	productRpc productclient.Product,
	stockRpc stockclient.Stock,
	producer mq.Producer,
	timeoutConfig config.OrderTimeoutConfig,
) OrderService {
	node, _ := snowflake.NewNode(1)
	return &orderServiceImpl{
		repo:          repo,
		outboxRepo:    outboxRepo,
		productRpc:    productRpc,
		stockRpc:      stockRpc,
		producer:      producer,
		snowflakeNode: node,
		timeoutConfig: timeoutConfig,
	}
}

/*
  CreateOrder 创建订单（重构版：StockRPC + Outbox Pattern）

  业务流程：
  1. 校验商品信息（ProductRPC）
  2. Redis Lua 原子扣减库存（StockRPC，独立库存服务）
  3. 生成订单号（Snowflake）
  4. 构建 Outbox 出站消息（order.created + order.delay.check）
  5. MySQL 单事务：扣减 MySQL 库存 + 插入订单 + 写入 Outbox 消息
  6. 若事务失败 → 回滚 Redis 库存
  7. 返回订单号

  原子性保证：
  - DB 事务保证 order + outbox 要么全部成功，要么全部失败
  - StockRPC 扣减失败时不会进入 DB 事务
  - DB 事务失败时回滚 StockRPC 的 Redis 扣减
  - Outbox Worker 异步将消息投递到 MQ，保证消息不丢
*/
func (s *orderServiceImpl) CreateOrder(ctx context.Context, req *pb.CreateOrderReq) (*pb.CreateOrderResp, error) {
	startTime := time.Now()
	defer func() {
		metrics.OrderCreateDuration.WithLabelValues().Observe(time.Since(startTime).Seconds())
	}()

	// 1. 校验商品信息
	prodResp, err := s.productRpc.GetProduct(ctx, &productclient.GetProductReq{Id: req.ProductId})
	if err != nil || prodResp.Product == nil {
		metrics.OrderCreateTotal.WithLabelValues("fail").Inc()
		return nil, ErrProductInvalid
	}

	// 2. Redis Lua 原子扣减库存（通过独立 Stock 服务）
	deductResp, err := s.stockRpc.DeductStock(ctx, &stockclient.DeductStockReq{
		ProductId: req.ProductId,
		Count:     req.Count,
	})
	if err != nil {
		metrics.OrderCreateTotal.WithLabelValues("fail").Inc()
		log.Printf("StockRPC.DeductStock 调用失败: %v", err)
		return nil, errors.New("系统繁忙，请稍后再试")
	}
	if !deductResp.Success {
		metrics.OrderCreateTotal.WithLabelValues("fail").Inc()
		return nil, ErrStockNotEnough
	}

	// 3. 生成订单号和过期时间
	orderNo := fmt.Sprintf("ORD%d", s.snowflakeNode.Generate().Int64())
	expireMinutes := s.timeoutConfig.OrderExpireMinutes
	if expireMinutes <= 0 {
		expireMinutes = 30
	}
	expireTime := time.Now().Add(time.Duration(expireMinutes) * time.Minute)
	totalAmount := prodResp.Product.Price * int64(req.Count)

	// 4. 构建订单实体
	newOrder := &repo.Order{
		OrderNo:     orderNo,
		UserID:      req.UserId,
		ProductID:   req.ProductId,
		Count:       req.Count,
		TotalAmount: totalAmount,
		Status:      int8(repo.OrderStatusPending),
	}

	// 5. 构建 Outbox 出站消息
	createMsgPayload, _ := json.Marshal(mq.OrderMsg{
		OrderNo:     orderNo,
		UserID:      req.UserId,
		ProductID:   req.ProductId,
		Count:       req.Count,
		TotalAmount: totalAmount,
		ExpireTime:  expireTime.Unix(),
	})

	delayMsgPayload, _ := json.Marshal(mq.DelayOrderMsg{
		OrderNo:    orderNo,
		ProductID:  req.ProductId,
		Count:      req.Count,
		CreateTime: time.Now(),
		ExpireTime: expireTime,
	})

	outboxRecords := []*repo.OutboxRecord{
		{
			MessageType: repo.OutboxTypeOrderCreated,
			Payload:     string(createMsgPayload),
			Status:      repo.OutboxStatusPending,
			NextRetryAt: time.Now(),
		},
		{
			MessageType: repo.OutboxTypeOrderDelay,
			Payload:     string(delayMsgPayload),
			Status:      repo.OutboxStatusPending,
			NextRetryAt: time.Now().Add(time.Duration(expireMinutes) * time.Minute),
		},
	}

	// 6. MySQL 单事务：扣减库存 + 插入订单 + 写入 Outbox 消息
	err = s.repo.CreateOrderWithOutboxTx(ctx, newOrder, expireTime, req.Count, outboxRecords)
	if err != nil {
		metrics.OrderCreateTotal.WithLabelValues("fail").Inc()
		log.Printf("CreateOrderWithOutboxTx 失败，回滚Redis库存: OrderNo=%s, Err=%v", orderNo, err)
		if _, rollbackErr := s.stockRpc.RollbackStock(ctx, &stockclient.RollbackStockReq{
			ProductId: req.ProductId,
			Count:     req.Count,
		}); rollbackErr != nil {
			log.Printf("严重: Redis库存回滚失败! OrderNo=%s, ProductID=%d, Count=%d, Err=%v",
				orderNo, req.ProductId, req.Count, rollbackErr)
		}
		return nil, errors.New("系统拥挤，请稍后再试")
	}

	metrics.OrderCreateTotal.WithLabelValues("success").Inc()
	metrics.MQPublishTotal.WithLabelValues("outbox", "success").Add(2) // 两条outbox消息

	log.Printf("订单创建成功: OrderNo=%s, UserID=%d, ProductID=%d, Count=%d, Amount=%d",
		orderNo, req.UserId, req.ProductId, req.Count, totalAmount)

	return &pb.CreateOrderResp{
		OrderNo:   orderNo,
		ExpireTime: expireTime.Unix(),
	}, nil
}

var (
	ErrOrderNotFound        = errors.New("订单不存在")
	ErrOrderAlreadyConfirmed = errors.New("订单已确认")
)

func (s *orderServiceImpl) ConfirmPay(ctx context.Context, req *pb.ConfirmPayReq) (*pb.ConfirmPayResp, error) {
	order, err := s.repo.GetOrderByNo(ctx, req.OrderNo)
	if err != nil {
		if errors.Is(err, repo.ErrOrderNotFound) {
			return &pb.ConfirmPayResp{Success: false, Message: "订单不存在"}, nil
		}
		return nil, err
	}

	if order.Status != 0 {
		return &pb.ConfirmPayResp{Success: false, Message: "订单状态不允许确认"}, nil
	}

	err = s.repo.UpdateOrderStatus(ctx, req.OrderNo, 1)
	if err != nil {
		return nil, err
	}

	return &pb.ConfirmPayResp{Success: true, Message: "支付确认成功"}, nil
}

func (s *orderServiceImpl) ListOrder(ctx context.Context, req *pb.ListOrderReq) (*pb.ListOrderResp, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	orders, total, err := s.repo.ListOrdersByUser(ctx, req.UserId, req.Page, req.PageSize, int8(req.Status))
	if err != nil {
		return nil, err
	}

	// 批量获取商品信息（消除 N+1 查询）
	productIDs := make([]int64, len(orders))
	for i, o := range orders {
		productIDs[i] = o.ProductID
	}
	productNameMap := make(map[int64]string)
	for _, pid := range productIDs {
		if prodResp, err := s.productRpc.GetProduct(ctx, &productclient.GetProductReq{Id: pid}); err == nil && prodResp.Product != nil {
			productNameMap[pid] = prodResp.Product.Name
		}
	}

	var items []*pb.OrderItem
	for _, order := range orders {
		item := &pb.OrderItem{
			Id:          order.ID,
			OrderNo:     order.OrderNo,
			UserId:      order.UserID,
			ProductId:   order.ProductID,
			ProductName: productNameMap[order.ProductID],
			Count:       order.Count,
			TotalAmount: order.TotalAmount,
			Status:      int32(order.Status),
			StatusText:  repo.OrderStatus(order.Status).StatusText(),
			CreateTime:  order.CreateTime.Unix(),
		}
		if order.ExpireTime != nil {
			item.ExpireTime = order.ExpireTime.Unix()
		}
		items = append(items, item)
	}

	return &pb.ListOrderResp{
		Orders:   items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *orderServiceImpl) GetOrderDetail(ctx context.Context, req *pb.GetOrderDetailReq) (*pb.GetOrderDetailResp, error) {
	order, err := s.repo.GetOrderByNo(ctx, req.OrderNo)
	if err != nil {
		if errors.Is(err, repo.ErrOrderNotFound) {
			return nil, errors.New("订单不存在")
		}
		return nil, err
	}

	if order.UserID != req.UserId {
		return nil, errors.New("无权访问此订单")
	}

	resp := &pb.GetOrderDetailResp{
		Id:          order.ID,
		OrderNo:     order.OrderNo,
		UserId:      order.UserID,
		ProductId:   order.ProductID,
		Count:       order.Count,
		TotalAmount: order.TotalAmount,
		Status:      int32(order.Status),
		StatusText:  repo.OrderStatus(order.Status).StatusText(),
		CreateTime:  order.CreateTime.Unix(),
	}

	if order.ExpireTime != nil {
		resp.ExpireTime = order.ExpireTime.Unix()
	} else {
		resp.ExpireTime = order.CreateTime.Add(30 * time.Minute).Unix()
	}

	if prodResp, err := s.productRpc.GetProduct(ctx, &productclient.GetProductReq{Id: order.ProductID}); err == nil && prodResp.Product != nil {
		resp.ProductName = prodResp.Product.Name
		resp.ProductDesc = prodResp.Product.Desc
	}

	return resp, nil
}

func (s *orderServiceImpl) CancelOrder(ctx context.Context, req *pb.CancelOrderReq) (*pb.CancelOrderResp, error) {
	order, err := s.repo.GetOrderByNo(ctx, req.OrderNo)
	if err != nil {
		if errors.Is(err, repo.ErrOrderNotFound) {
			return &pb.CancelOrderResp{Success: false, Message: "订单不存在"}, nil
		}
		return nil, err
	}

	if order.UserID != req.UserId {
		return &pb.CancelOrderResp{Success: false, Message: "无权取消此订单"}, nil
	}

	// 取消订单（更新状态 + 回滚库存，在事务中）
	err = s.repo.CancelOrderTx(ctx, req.OrderNo, req.UserId, order.ProductID, order.Count)
	if err != nil {
		return &pb.CancelOrderResp{Success: false, Message: err.Error()}, nil
	}

	// 异步回滚 Redis 库存（通过 StockRPC）
	go func() {
		bgCtx := context.Background()
		if _, err := s.stockRpc.RollbackStock(bgCtx, &stockclient.RollbackStockReq{
			ProductId: order.ProductID,
			Count:     order.Count,
		}); err != nil {
			log.Printf("取消订单后Redis库存回滚失败: OrderNo=%s, Err=%v", order.OrderNo, err)
		}
	}()

	return &pb.CancelOrderResp{Success: true, Message: "订单取消成功"}, nil
}
