package payment

import (
	"context"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	paymentpb "ecommerce-demo/app/payment/pb"
	"ecommerce-demo/common/ctxutil"

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

func (l *CreatePayLogic) CreatePay(req *types.CreatePayReq) (*types.CreatePayResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.PaymentRpc.CreatePay(l.ctx, &paymentpb.CreatePayReq{
		OrderNo:    req.OrderNo,
		UserId:     userId,
		Amount:     req.Amount,
		PayChannel: req.PayChannel,
	})

	if err != nil {
		return nil, err
	}

	return &types.CreatePayResp{
		PaymentNo:  resp.PaymentNo,
		QrCode:     resp.QrCode,
		ExpireTime: resp.ExpireTime,
	}, nil
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

func (l *GetPayStatusLogic) GetPayStatus(paymentNo string) (*types.GetPayStatusResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.PaymentRpc.GetPayStatus(l.ctx, &paymentpb.GetPayStatusReq{
		PaymentNo: paymentNo,
		UserId:    userId,
	})

	if err != nil {
		return nil, err
	}

	return &types.GetPayStatusResp{
		PaymentNo:  resp.PaymentNo,
		Status:     resp.Status,
		StatusText: resp.StatusText,
		Amount:     resp.Amount,
		OrderNo:    resp.OrderNo,
	}, nil
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

func (l *CancelPayLogic) CancelPay(paymentNo string) (*types.CancelPayResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	resp, err := l.svcCtx.PaymentRpc.CancelPay(l.ctx, &paymentpb.CancelPayReq{
		PaymentNo: paymentNo,
		UserId:    userId,
	})

	if err != nil {
		return nil, err
	}

	return &types.CancelPayResp{
		Message: resp.Message,
	}, nil
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

func (l *ListPayLogic) ListPay(page, pageSize int32) (*types.ListPayResp, error) {
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	resp, err := l.svcCtx.PaymentRpc.ListPay(l.ctx, &paymentpb.ListPayReq{
		UserId:   userId,
		Page:     page,
		PageSize: pageSize,
	})

	if err != nil {
		return nil, err
	}

	var payments []types.PaymentInfo
	for _, p := range resp.Payments {
		payments = append(payments, types.PaymentInfo{
			Id:         p.Id,
			PaymentNo:  p.PaymentNo,
			OrderNo:    p.OrderNo,
			Amount:     p.Amount,
			Status:     p.Status,
			StatusText: p.StatusText,
			PayChannel: p.PayChannel,
			PayTime:    p.PayTime,
			ExpireTime: p.ExpireTime,
			CreatedAt:  p.CreatedAt,
		})
	}

	return &types.ListPayResp{
		Payments: payments,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}
