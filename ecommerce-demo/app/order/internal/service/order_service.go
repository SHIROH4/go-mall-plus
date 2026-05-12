package service

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    "ecommerce-demo/app/order/internal/config"
    "ecommerce-demo/app/order/internal/mq"
    "ecommerce-demo/app/order/internal/repo"
    "ecommerce-demo/app/order/pb"
    productclient "ecommerce-demo/app/product/product"

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
    productRpc    productclient.Product
    producer      mq.Producer
    snowflakeNode *snowflake.Node
    // [新增] 超时配置
    timeoutConfig config.OrderTimeoutConfig
}

func NewOrderService(
    repo repo.OrderRepo,
    productRpc productclient.Product,
    producer mq.Producer,
    timeoutConfig config.OrderTimeoutConfig,
) OrderService {
    node, _ := snowflake.NewNode(1)
    return &orderServiceImpl{
        repo:          repo,
        productRpc:    productRpc,
        producer:      producer,
        snowflakeNode: node,
        timeoutConfig: timeoutConfig,
    }
}

/*
  CreateOrder 创建订单

  业务流程：
  1. 校验商品信息
  2. Redis Lua 原子扣减库存（前置拦截）
  3. 生成订单号
  4. 发送 MQ 消息（异步落库）
  5. [新增] 发送延迟 MQ 消息（超时检查）
  6. [修改] MQ 发送失败则回滚 Redis 库存

  幂等保证：
  - 同一订单号不会重复创建（由雪花ID保证唯一性）
*/
func (s *orderServiceImpl) CreateOrder(ctx context.Context, req *pb.CreateOrderReq) (*pb.CreateOrderResp, error) {
    // 1. 校验商品信息
    prodResp, err := s.productRpc.GetProduct(ctx, &productclient.GetProductReq{Id: req.ProductId})
    if err != nil || prodResp.Product == nil {
        return nil, ErrProductInvalid
    }

    // 2. Redis Lua 原子扣减库存
    success, err := s.repo.DeductStockCache(ctx, req.ProductId, req.Count)
    if err != nil {
        return nil, err
    }
    if !success {
        return nil, ErrStockNotEnough
    }

    // 3. 生成订单号和过期时间
    orderNo := fmt.Sprintf("ORD%d", s.snowflakeNode.Generate().Int64())
    expireMinutes := s.timeoutConfig.OrderExpireMinutes
    if expireMinutes <= 0 {
        expireMinutes = 30 // 默认30分钟
    }
    expireTime := time.Now().Add(time.Duration(expireMinutes) * time.Minute)
    totalAmount := prodResp.Product.Price * int64(req.Count)

    // 4. 构建异步落库消息
    msg := &mq.OrderMsg{
        OrderNo:     orderNo,
        UserID:      req.UserId,
        ProductID:   req.ProductId,
        Count:       req.Count,
        TotalAmount: totalAmount,
        ExpireTime:  expireTime.Unix(), // [新增] 传递过期时间
    }

    // 5. 发送异步落库消息
    if err := s.producer.PublishOrder(ctx, msg); err != nil {
        // 落库消息发送失败，回滚 Redis 库存
        s.repo.RollbackStockCache(ctx, req.ProductId, req.Count)
        return nil, errors.New("系统拥挤，请稍后再试")
    }

    // 6. [新增] 发送延迟超时检查消息
    delayMsg := mq.BuildDelayOrderMsg(orderNo, req.ProductId, req.Count, expireTime)
    if err := s.producer.PublishDelayOrder(ctx, delayMsg, expireMinutes); err != nil {
        // 延迟消息发送失败不影响主流程
        // 定时扫描兜底任务会处理这些订单
        log.Printf("⚠️ 延迟消息发送失败（非致命）: OrderNo=%s, Err=%v", orderNo, err)
    }

    return &pb.CreateOrderResp{
        OrderNo: orderNo,
    }, nil
}

/*
  ConfirmPay 支付确认

  当用户完成支付后，支付服务回调此接口

  业务流程：
  1. 查询订单
  2. 校验订单状态（必须是待支付）
  3. 更新状态为已支付

  注意：此方法不处理库存，因为库存已经在下单时扣减了
*/
var (
    ErrOrderNotFound         = errors.New("订单不存在")
    ErrOrderAlreadyConfirmed  = errors.New("订单已确认")
)

func (s *orderServiceImpl) ConfirmPay(ctx context.Context, req *pb.ConfirmPayReq) (*pb.ConfirmPayResp, error) {
    order, err := s.repo.GetOrderByNo(ctx, req.OrderNo)
    if err != nil {
        if errors.Is(err, repo.ErrOrderNotFound) {
            return &pb.ConfirmPayResp{
                Success: false,
                Message: "订单不存在",
            }, nil
        }
        return nil, err
    }

    if order.Status != 0 {
        return &pb.ConfirmPayResp{
            Success: false,
            Message: "订单状态不允许确认",
        }, nil
    }

    err = s.repo.UpdateOrderStatus(ctx, req.OrderNo, 1)
    if err != nil {
        return nil, err
    }

    return &pb.ConfirmPayResp{
        Success: true,
        Message: "支付确认成功",
    }, nil
}

/*
  ListOrder 订单列表

  支持分页和状态筛选
*/
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

    var items []*pb.OrderItem
    for _, order := range orders {
        item := &pb.OrderItem{
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

        // [新增] 过期时间
        if order.ExpireTime != nil {
            item.ExpireTime = order.ExpireTime.Unix()
        }

        // 获取商品名称
        if prodResp, err := s.productRpc.GetProduct(ctx, &productclient.GetProductReq{Id: order.ProductID}); err == nil && prodResp.Product != nil {
            item.ProductName = prodResp.Product.Name
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

/*
  GetOrderDetail 订单详情

  返回订单详细信息，包括商品快照（未来改造点）
*/
func (s *orderServiceImpl) GetOrderDetail(ctx context.Context, req *pb.GetOrderDetailReq) (*pb.GetOrderDetailResp, error) {
    order, err := s.repo.GetOrderByNo(ctx, req.OrderNo)
    if err != nil {
        if errors.Is(err, repo.ErrOrderNotFound) {
            return nil, errors.New("订单不存在")
        }
        return nil, err
    }

    // 验证订单归属
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

    // [修改] 过期时间从数据库读取，而非硬编码
    if order.ExpireTime != nil {
        resp.ExpireTime = order.ExpireTime.Unix()
    } else {
        // 兼容旧数据：没有过期时间字段时，使用创建时间+30分钟
        resp.ExpireTime = order.CreateTime.Add(30 * time.Minute).Unix()
    }

    // 获取商品详情
    if prodResp, err := s.productRpc.GetProduct(ctx, &productclient.GetProductReq{Id: order.ProductID}); err == nil && prodResp.Product != nil {
        resp.ProductName = prodResp.Product.Name
        resp.ProductDesc = prodResp.Product.Desc
    }

    return resp, nil
}

/*
  CancelOrder 取消订单

  用户主动取消订单

  业务流程：
  1. 查询订单并校验归属
  2. 执行取消事务（更新状态 + 回滚库存）
*/
func (s *orderServiceImpl) CancelOrder(ctx context.Context, req *pb.CancelOrderReq) (*pb.CancelOrderResp, error) {
    order, err := s.repo.GetOrderByNo(ctx, req.OrderNo)
    if err != nil {
        if errors.Is(err, repo.ErrOrderNotFound) {
            return &pb.CancelOrderResp{
                Success: false,
                Message: "订单不存在",
            }, nil
        }
        return nil, err
    }

    // 验证订单归属
    if order.UserID != req.UserId {
        return &pb.CancelOrderResp{
            Success: false,
            Message: "无权取消此订单",
        }, nil
    }

    // 执行取消事务
    err = s.repo.CancelOrderTx(ctx, req.OrderNo, req.UserId, order.ProductID, order.Count)
    if err != nil {
        return &pb.CancelOrderResp{
            Success: false,
            Message: err.Error(),
        }, nil
    }

    return &pb.CancelOrderResp{
        Success: true,
        Message: "订单取消成功",
    }, nil
}
