package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	orderclient "ecommerce-demo/app/order/order"
	"ecommerce-demo/app/payment/internal/model"
	"ecommerce-demo/app/payment/internal/repo"
	"ecommerce-demo/app/payment/pb"
	"ecommerce-demo/common/metrics"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	DefaultPayExpireMinutes = 30 // 默认支付过期时间30分钟
	MaxPayExpireMinutes     = 120
)

var (
	ErrPaymentNotFound    = errors.New("支付单不存在")
	ErrPaymentExpired     = errors.New("支付单已过期")
	ErrPaymentCancelled   = errors.New("支付单已取消")
	ErrPaymentAlreadyPaid = errors.New("支付单已支付")
	ErrOrderAlreadyPaid   = errors.New("订单已支付")
	ErrInvalidPayChannel  = errors.New("无效的支付渠道")
)

type PaymentService interface {
	CreatePay(ctx context.Context, req *pb.CreatePayReq) (*pb.CreatePayResp, error)
	GetPayStatus(ctx context.Context, req *pb.GetPayStatusReq) (*pb.GetPayStatusResp, error)
	CancelPay(ctx context.Context, req *pb.CancelPayReq) (*pb.CancelPayResp, error)
	PayCallback(ctx context.Context, req *pb.PayCallbackReq) (*pb.PayCallbackResp, error)
	ListPay(ctx context.Context, req *pb.ListPayReq) (*pb.ListPayResp, error)
	CloseExpiredPay(ctx context.Context, req *pb.CloseExpiredPayReq) (*pb.CloseExpiredPayResp, error)
}

type paymentServiceImpl struct {
	repo      repo.PaymentRepo
	orderRpc  orderclient.Order
	expireMin int
	logx.Logger
}

func NewPaymentService(repo repo.PaymentRepo, orderRpc orderclient.Order) PaymentService {
	return &paymentServiceImpl{
		repo:      repo,
		orderRpc:  orderRpc,
		expireMin: DefaultPayExpireMinutes,
		Logger:    logx.WithContext(context.Background()),
	}
}

func NewPaymentServiceWithConfig(repo repo.PaymentRepo, orderRpc orderclient.Order, expireMin int) PaymentService {
	expire := expireMin
	if expire <= 0 {
		expire = DefaultPayExpireMinutes
	}
	if expire > MaxPayExpireMinutes {
		expire = MaxPayExpireMinutes
	}
	return &paymentServiceImpl{
		repo:      repo,
		orderRpc:  orderRpc,
		expireMin: expire,
		Logger:    logx.WithContext(context.Background()),
	}
}

func (s *paymentServiceImpl) CreatePay(ctx context.Context, req *pb.CreatePayReq) (*pb.CreatePayResp, error) {
	// 1. 检查是否已有支付单
	existing, _ := s.repo.GetByOrderNo(ctx, req.OrderNo)
	if existing != nil {
		switch existing.Status {
		case model.PaymentStatus_Paid:
			return nil, ErrPaymentAlreadyPaid
		case model.PaymentStatus_Pending:
			// 已存在待支付单，直接返回
			return s.buildPayResp(existing)
		case model.PaymentStatus_Cancelled, model.PaymentStatus_Expired:
			// 已取消或已过期，允许重新创建
		}
	}

	// 2. 验证支付渠道
	if !isValidPayChannel(req.PayChannel) {
		return nil, ErrInvalidPayChannel
	}

	// 3. 生成支付单号
	paymentNo := generatePaymentNo()

	// 4. 创建支付单
	now := time.Now()
	expireTime := now.Add(time.Duration(s.expireMin) * time.Minute)

	payment := &model.Payment{
		PaymentNo:  paymentNo,
		OrderNo:    req.OrderNo,
		UserID:     req.UserId,
		Amount:     req.Amount,
		Status:     model.PaymentStatus_Pending,
		PayChannel: req.PayChannel,
		ExpireTime: expireTime,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
		metrics.PaymentTotal.WithLabelValues("pending", req.PayChannel).Inc()
	}

	// 5. 生成模拟二维码（实际应调用第三方支付SDK）
	qrCode := generateQrCode(paymentNo)

	return &pb.CreatePayResp{
		PaymentNo:  paymentNo,
		QrCode:     qrCode,
		ExpireTime: expireTime.Unix(),
	}, nil
}

func (s *paymentServiceImpl) GetPayStatus(ctx context.Context, req *pb.GetPayStatusReq) (*pb.GetPayStatusResp, error) {
	payment, err := s.repo.GetByPaymentNo(ctx, req.PaymentNo)
	if err != nil {
		if errors.Is(err, repo.ErrPaymentNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}

	// 检查用户权限
	if payment.UserID != req.UserId {
		return nil, ErrPaymentNotFound
	}

	// 检查是否过期但状态未更新
	currentStatus := payment.Status
	if currentStatus == model.PaymentStatus_Pending && payment.IsExpired() {
		currentStatus = model.PaymentStatus_Expired
	}

	return &pb.GetPayStatusResp{
		PaymentNo:  payment.PaymentNo,
		Status:     int32(currentStatus),
		StatusText: currentStatus.String(),
		Amount:     payment.Amount,
		OrderNo:    payment.OrderNo,
	}, nil
}

func (s *paymentServiceImpl) CancelPay(ctx context.Context, req *pb.CancelPayReq) (*pb.CancelPayResp, error) {
	payment, err := s.repo.GetByPaymentNo(ctx, req.PaymentNo)
	if err != nil {
		if errors.Is(err, repo.ErrPaymentNotFound) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}

	// 检查用户权限
	if payment.UserID != req.UserId {
		return nil, ErrPaymentNotFound
	}

	// 检查状态
	if !payment.CanCancel() {
		if payment.Status == model.PaymentStatus_Paid {
			return nil, ErrPaymentAlreadyPaid
		}
		return nil, ErrPaymentCancelled
	}

	// 更新状态
	if err := s.repo.UpdateStatus(ctx, req.PaymentNo, model.PaymentStatus_Cancelled, ""); err != nil {
		return nil, err
	}

	return &pb.CancelPayResp{
		Message: "取消成功",
	}, nil
}

func (s *paymentServiceImpl) PayCallback(ctx context.Context, req *pb.PayCallbackReq) (*pb.PayCallbackResp, error) {
	// 1. 查询支付单
	payment, err := s.repo.GetByPaymentNo(ctx, req.PaymentNo)
	if err != nil {
		return &pb.PayCallbackResp{
			Success: false,
			Message: "支付单不存在",
		}, nil
	}

	// 2. 检查状态
	if payment.Status != model.PaymentStatus_Pending {
		return &pb.PayCallbackResp{
			Success: true,
			Message: "已处理",
		}, nil
	}

	// 3. 检查过期
	if payment.IsExpired() {
		s.repo.UpdateStatus(ctx, req.PaymentNo, model.PaymentStatus_Expired, req.CallbackData)

		metrics.PaymentTotal.WithLabelValues("expired", payment.PayChannel).Inc()
		return &pb.PayCallbackResp{
			Success: false,
			Message: "支付单已过期",
		}, nil
	}

	// 4. 更新为已支付（使用事务确保幂等）
	if err := s.repo.UpdateStatusWithTx(ctx, req.PaymentNo, model.PaymentStatus_Paid, req.CallbackData); err != nil {
		return &pb.PayCallbackResp{
			Success: false,
			Message: "更新失败",
		}, nil
	}

	// 5. 通知订单服务更新状态（异步）
	go func() {
		if s.orderRpc != nil {
			// 调用订单服务确认支付
			_, _ = s.orderRpc.ConfirmPay(context.Background(), &orderclient.ConfirmPayReq{
				OrderNo:    payment.OrderNo,
				PaymentNo:  payment.PaymentNo,
				PayChannel: req.PayChannel,
			})
		}
	}()

	metrics.PaymentTotal.WithLabelValues("success", req.PayChannel).Inc()
	metrics.PaymentAmountTotal.WithLabelValues(req.PayChannel).Add(float64(payment.Amount))

	return &pb.PayCallbackResp{
		Success: true,
		Message: "支付成功",
	}, nil
}

func (s *paymentServiceImpl) ListPay(ctx context.Context, req *pb.ListPayReq) (*pb.ListPayResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	payments, total, err := s.repo.ListByUserID(ctx, req.UserId, page, pageSize)
	if err != nil {
		return nil, err
	}

	var pbPayments []*pb.PaymentInfo
	for _, p := range payments {
		pbPayments = append(pbPayments, s.toPbPayment(p))
	}

	return &pb.ListPayResp{
		Payments: pbPayments,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *paymentServiceImpl) CloseExpiredPay(ctx context.Context, req *pb.CloseExpiredPayReq) (*pb.CloseExpiredPayResp, error) {
	// 批量查询过期支付单
	expiredPayments, err := s.repo.GetExpiredPayments(ctx, 100)
	if err != nil {
		return nil, err
	}

	if len(expiredPayments) == 0 {
		return &pb.CloseExpiredPayResp{ClosedCount: 0}, nil
	}

	// 提取支付单号
	paymentNos := make([]string, len(expiredPayments))
	for i, p := range expiredPayments {
		paymentNos[i] = p.PaymentNo
	}

	// 批量关闭
	closedCount, err := s.repo.CloseExpiredPayments(ctx, paymentNos)
	if err != nil {
		return nil, err
	}

	logx.Infof("关闭了 %d 个过期支付单", closedCount)

	return &pb.CloseExpiredPayResp{ClosedCount: int32(closedCount)}, nil
}

func (s *paymentServiceImpl) buildPayResp(payment *model.Payment) (*pb.CreatePayResp, error) {
	return &pb.CreatePayResp{
		PaymentNo:  payment.PaymentNo,
		QrCode:     generateQrCode(payment.PaymentNo),
		ExpireTime: payment.ExpireTime.Unix(),
	}, nil
}

func (s *paymentServiceImpl) toPbPayment(p *model.Payment) *pb.PaymentInfo {
	status := p.Status
	statusText := s.PaymentStatusToString(status)
	if status == model.PaymentStatus_Pending && p.IsExpired() {
		status = model.PaymentStatus_Expired
		statusText = "已超时"
	}

	var payTime int64
	if p.PayTime != nil {
		payTime = p.PayTime.Unix()
	}

	return &pb.PaymentInfo{
		Id:         p.ID,
		PaymentNo:  p.PaymentNo,
		OrderNo:    p.OrderNo,
		UserId:     p.UserID,
		Amount:     p.Amount,
		Status:     int32(status),
		StatusText: statusText,
		PayChannel: p.PayChannel,
		PayTime:    payTime,
		ExpireTime: p.ExpireTime.Unix(),
		CreatedAt:  p.CreatedAt.Unix(),
		UpdatedAt:  p.UpdatedAt.Unix(),
	}
}

func (s *paymentServiceImpl) PaymentStatusToString(status model.PaymentStatus) string {
	switch status {
	case model.PaymentStatus_Pending:
		return "待支付"
	case model.PaymentStatus_Paid:
		return "已支付"
	case model.PaymentStatus_Cancelled:
		return "已取消"
	case model.PaymentStatus_Expired:
		return "已超时"
	default:
		return "未知"
	}
}

func isValidPayChannel(channel string) bool {
	return channel == "alipay" || channel == "wechat"
}

func generatePaymentNo() string {
	timestamp := time.Now().Unix()
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	return fmt.Sprintf("PAY%d%s", timestamp, hex.EncodeToString(randBytes))
}

func generateQrCode(paymentNo string) string {
	return fmt.Sprintf("https://pay.example.com/qrcode/%s", paymentNo)
}
