package service

import (
	"context"
	"testing"
	"time"

	"ecommerce-demo/app/payment/internal/model"
	"ecommerce-demo/app/payment/internal/repo"
	"ecommerce-demo/app/payment/pb"
)

type mockPaymentRepo struct {
	payments map[string]*model.Payment
}

func newMockPaymentRepo() *mockPaymentRepo {
	return &mockPaymentRepo{
		payments: make(map[string]*model.Payment),
	}
}

func (m *mockPaymentRepo) Create(ctx context.Context, payment *model.Payment) error {
	m.payments[payment.PaymentNo] = payment
	return nil
}

func (m *mockPaymentRepo) GetByPaymentNo(ctx context.Context, paymentNo string) (*model.Payment, error) {
	if p, ok := m.payments[paymentNo]; ok {
		return p, nil
	}
	return nil, repo.ErrPaymentNotFound
}

func (m *mockPaymentRepo) GetByOrderNo(ctx context.Context, orderNo string) (*model.Payment, error) {
	for _, p := range m.payments {
		if p.OrderNo == orderNo {
			return p, nil
		}
	}
	return nil, repo.ErrPaymentNotFound
}

func (m *mockPaymentRepo) UpdateStatus(ctx context.Context, paymentNo string, status int32, callbackData string) error {
	if p, ok := m.payments[paymentNo]; ok {
		p.Status = status
		return nil
	}
	return repo.ErrPaymentNotFound
}

func (m *mockPaymentRepo) UpdateStatusWithTx(ctx context.Context, paymentNo string, status int32, callbackData string) error {
	return m.UpdateStatus(ctx, paymentNo, status, callbackData)
}

func (m *mockPaymentRepo) ListByUserID(ctx context.Context, userID int64, page, pageSize int32) ([]*model.Payment, int32, error) {
	var result []*model.Payment
	for _, p := range m.payments {
		if p.UserID == userID {
			result = append(result, p)
		}
	}
	return result, int32(len(result)), nil
}

func (m *mockPaymentRepo) GetExpiredPayments(ctx context.Context, limit int) ([]*model.Payment, error) {
	var result []*model.Payment
	for _, p := range m.payments {
		if p.Status == model.PaymentStatus_Pending && p.ExpireTime.Before(time.Now()) {
			result = append(result, p)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockPaymentRepo) CloseExpiredPayments(ctx context.Context, paymentNos []string) (int64, error) {
	var count int64
	for _, no := range paymentNos {
		if p, ok := m.payments[no]; ok && p.Status == model.PaymentStatus_Pending {
			p.Status = model.PaymentStatus_Expired
			count++
		}
	}
	return count, nil
}

func TestCreatePay(t *testing.T) {
	repo := newMockPaymentRepo()
	svc := NewPaymentService(repo, nil)

	ctx := context.Background()
	req := &pb.CreatePayReq{
		OrderNo:    "ORD123456",
		UserId:     1001,
		Amount:     599900,
		PayChannel: "alipay",
	}

	resp, err := svc.CreatePay(ctx, req)
	if err != nil {
		t.Fatalf("CreatePay failed: %v", err)
	}

	if resp.PaymentNo == "" {
		t.Error("PaymentNo should not be empty")
	}

	if resp.QrCode == "" {
		t.Error("QrCode should not be empty")
	}

	if resp.ExpireTime == 0 {
		t.Error("ExpireTime should not be zero")
	}
}

func TestGetPayStatus(t *testing.T) {
	repo := newMockPaymentRepo()
	svc := NewPaymentService(repo, nil)

	ctx := context.Background()

	// 先创建支付单
	createResp, _ := svc.CreatePay(ctx, &pb.CreatePayReq{
		OrderNo:    "ORD123456",
		UserId:     1001,
		Amount:     599900,
		PayChannel: "alipay",
	})

	// 查询状态
	status, err := svc.GetPayStatus(ctx, &pb.GetPayStatusReq{
		PaymentNo: createResp.PaymentNo,
		UserId:    1001,
	})

	if err != nil {
		t.Fatalf("GetPayStatus failed: %v", err)
	}

	if status.Status != 0 {
		t.Errorf("Expected status 0 (pending), got %d", status.Status)
	}

	if status.Amount != 599900 {
		t.Errorf("Expected amount 599900, got %d", status.Amount)
	}
}

func TestCancelPay(t *testing.T) {
	repo := newMockPaymentRepo()
	svc := NewPaymentService(repo, nil)

	ctx := context.Background()

	// 先创建支付单
	createResp, _ := svc.CreatePay(ctx, &pb.CreatePayReq{
		OrderNo:    "ORD123456",
		UserId:     1001,
		Amount:     599900,
		PayChannel: "alipay",
	})

	// 取消支付
	cancelResp, err := svc.CancelPay(ctx, &pb.CancelPayReq{
		PaymentNo: createResp.PaymentNo,
		UserId:    1001,
	})

	if err != nil {
		t.Fatalf("CancelPay failed: %v", err)
	}

	if cancelResp.Message != "取消成功" {
		t.Errorf("Expected message '取消成功', got '%s'", cancelResp.Message)
	}

	// 验证状态已更新
	status, _ := svc.GetPayStatus(ctx, &pb.GetPayStatusReq{
		PaymentNo: createResp.PaymentNo,
		UserId:    1001,
	})

	if status.Status != 2 {
		t.Errorf("Expected status 2 (cancelled), got %d", status.Status)
	}
}

func TestPayCallback(t *testing.T) {
	repo := newMockPaymentRepo()
	svc := NewPaymentService(repo, nil)

	ctx := context.Background()

	// 先创建支付单
	createResp, _ := svc.CreatePay(ctx, &pb.CreatePayReq{
		OrderNo:    "ORD123456",
		UserId:     1001,
		Amount:     599900,
		PayChannel: "alipay",
	})

	// 模拟支付回调
	callbackResp, err := svc.PayCallback(ctx, &pb.PayCallbackReq{
		PaymentNo:   createResp.PaymentNo,
		PayChannel:  "alipay",
		CallbackData: `{"trade_no":"ALIPAY123"}`,
	})

	if err != nil {
		t.Fatalf("PayCallback failed: %v", err)
	}

	if !callbackResp.Success {
		t.Error("Callback should succeed")
	}

	// 验证状态已更新
	status, _ := svc.GetPayStatus(ctx, &pb.GetPayStatusReq{
		PaymentNo: createResp.PaymentNo,
		UserId:    1001,
	})

	if status.Status != 1 {
		t.Errorf("Expected status 1 (paid), got %d", status.Status)
	}
}

func TestCannotPayTwice(t *testing.T) {
	repo := newMockPaymentRepo()
	svc := NewPaymentService(repo, nil)

	ctx := context.Background()

	// 先创建支付单
	createResp, _ := svc.CreatePay(ctx, &pb.CreatePayReq{
		OrderNo:    "ORD123456",
		UserId:     1001,
		Amount:     599900,
		PayChannel: "alipay",
	})

	// 第一次回调
	svc.PayCallback(ctx, &pb.PayCallbackReq{
		PaymentNo:  createResp.PaymentNo,
		PayChannel: "alipay",
	})

	// 第二次回调（应该幂等处理）
	callbackResp, _ := svc.PayCallback(ctx, &pb.PayCallbackReq{
		PaymentNo:  createResp.PaymentNo,
		PayChannel: "alipay",
	})

	if !callbackResp.Success {
		t.Error("Second callback should still succeed (idempotent)")
	}
}

func TestListPay(t *testing.T) {
	repo := newMockPaymentRepo()
	svc := NewPaymentService(repo, nil)

	ctx := context.Background()
	userID := int64(1001)

	// 创建多个支付单
	for i := 0; i < 3; i++ {
		svc.CreatePay(ctx, &pb.CreatePayReq{
			OrderNo:    "ORD" + string(rune('A'+i)),
			UserId:     userID,
			Amount:     10000,
			PayChannel: "alipay",
		})
	}

	// 查询列表
	listResp, err := svc.ListPay(ctx, &pb.ListPayReq{
		UserId:   userID,
		Page:     1,
		PageSize: 10,
	})

	if err != nil {
		t.Fatalf("ListPay failed: %v", err)
	}

	if listResp.Total != 3 {
		t.Errorf("Expected 3 payments, got %d", listResp.Total)
	}
}

func TestInvalidPayChannel(t *testing.T) {
	repo := newMockPaymentRepo()
	svc := NewPaymentService(repo, nil)

	ctx := context.Background()

	_, err := svc.CreatePay(ctx, &pb.CreatePayReq{
		OrderNo:    "ORD123456",
		UserId:     1001,
		Amount:     599900,
		PayChannel: "invalid",
	})

	if err != ErrInvalidPayChannel {
		t.Errorf("Expected ErrInvalidPayChannel, got %v", err)
	}
}
