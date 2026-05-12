package logic

import (
	"context"

	"ecommerce-demo/app/payment/internal/svc"
	"ecommerce-demo/app/payment/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreatePayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePayLogic {
	return &CreatePayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreatePayLogic) CreatePay(in *pb.CreatePayReq) (*pb.CreatePayResp, error) {
	return l.svcCtx.PaymentService.CreatePay(l.ctx, in)
}

type GetPayStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPayStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPayStatusLogic {
	return &GetPayStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPayStatusLogic) GetPayStatus(in *pb.GetPayStatusReq) (*pb.GetPayStatusResp, error) {
	return l.svcCtx.PaymentService.GetPayStatus(l.ctx, in)
}

type CancelPayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelPayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelPayLogic {
	return &CancelPayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelPayLogic) CancelPay(in *pb.CancelPayReq) (*pb.CancelPayResp, error) {
	return l.svcCtx.PaymentService.CancelPay(l.ctx, in)
}

type PayCallbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPayCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PayCallbackLogic {
	return &PayCallbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PayCallbackLogic) PayCallback(in *pb.PayCallbackReq) (*pb.PayCallbackResp, error) {
	return l.svcCtx.PaymentService.PayCallback(l.ctx, in)
}

type ListPayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPayLogic {
	return &ListPayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListPayLogic) ListPay(in *pb.ListPayReq) (*pb.ListPayResp, error) {
	return l.svcCtx.PaymentService.ListPay(l.ctx, in)
}

type CloseExpiredPayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCloseExpiredPayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CloseExpiredPayLogic {
	return &CloseExpiredPayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CloseExpiredPayLogic) CloseExpiredPay(in *pb.CloseExpiredPayReq) (*pb.CloseExpiredPayResp, error) {
	return l.svcCtx.PaymentService.CloseExpiredPay(l.ctx, in)
}
