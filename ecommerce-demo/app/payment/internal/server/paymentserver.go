package server

import (
	"context"

	"ecommerce-demo/app/payment/internal/logic"
	"ecommerce-demo/app/payment/internal/svc"
	"ecommerce-demo/app/payment/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type PaymentServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedPaymentServer
}

func NewPaymentServer(svcCtx *svc.ServiceContext) *PaymentServer {
	return &PaymentServer{svcCtx: svcCtx}
}

func (s *PaymentServer) CreatePay(ctx context.Context, in *pb.CreatePayReq) (*pb.CreatePayResp, error) {
	l := logic.NewCreatePayLogic(ctx, s.svcCtx)
	return l.CreatePay(in)
}

func (s *PaymentServer) GetPayStatus(ctx context.Context, in *pb.GetPayStatusReq) (*pb.GetPayStatusResp, error) {
	l := logic.NewGetPayStatusLogic(ctx, s.svcCtx)
	return l.GetPayStatus(in)
}

func (s *PaymentServer) CancelPay(ctx context.Context, in *pb.CancelPayReq) (*pb.CancelPayResp, error) {
	l := logic.NewCancelPayLogic(ctx, s.svcCtx)
	return l.CancelPay(in)
}

func (s *PaymentServer) PayCallback(ctx context.Context, in *pb.PayCallbackReq) (*pb.PayCallbackResp, error) {
	l := logic.NewPayCallbackLogic(ctx, s.svcCtx)
	return l.PayCallback(in)
}

func (s *PaymentServer) ListPay(ctx context.Context, in *pb.ListPayReq) (*pb.ListPayResp, error) {
	l := logic.NewListPayLogic(ctx, s.svcCtx)
	return l.ListPay(in)
}

func (s *PaymentServer) CloseExpiredPay(ctx context.Context, in *pb.CloseExpiredPayReq) (*pb.CloseExpiredPayResp, error) {
	l := logic.NewCloseExpiredPayLogic(ctx, s.svcCtx)
	return l.CloseExpiredPay(in)
}

func (s *PaymentServer) Ping(ctx context.Context) error {
	logx.Info("Payment RPC Server is alive")
	return nil
}
