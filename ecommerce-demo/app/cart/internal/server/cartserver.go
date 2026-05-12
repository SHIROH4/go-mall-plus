package server

import (
	"context"

	"ecommerce-demo/app/cart/internal/logic"
	"ecommerce-demo/app/cart/internal/svc"
	"ecommerce-demo/app/cart/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CartServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedCartServer
}

func NewCartServer(svcCtx *svc.ServiceContext) *CartServer {
	return &CartServer{svcCtx: svcCtx}
}

func (s *CartServer) AddCart(ctx context.Context, in *pb.AddCartReq) (*pb.AddCartResp, error) {
	l := logic.NewAddCartLogic(ctx, s.svcCtx)
	return l.AddCart(in)
}

func (s *CartServer) GetCart(ctx context.Context, in *pb.GetCartReq) (*pb.GetCartResp, error) {
	l := logic.NewGetCartLogic(ctx, s.svcCtx)
	return l.GetCart(in)
}

func (s *CartServer) UpdateCart(ctx context.Context, in *pb.UpdateCartReq) (*pb.UpdateCartResp, error) {
	l := logic.NewUpdateCartLogic(ctx, s.svcCtx)
	return l.UpdateCart(in)
}

func (s *CartServer) RemoveCart(ctx context.Context, in *pb.RemoveCartReq) (*pb.RemoveCartResp, error) {
	l := logic.NewRemoveCartLogic(ctx, s.svcCtx)
	return l.RemoveCart(in)
}

func (s *CartServer) ClearCart(ctx context.Context, in *pb.ClearCartReq) (*pb.ClearCartResp, error) {
	l := logic.NewClearCartLogic(ctx, s.svcCtx)
	return l.ClearCart(in)
}

func (s *CartServer) SelectCart(ctx context.Context, in *pb.SelectCartReq) (*pb.SelectCartResp, error) {
	l := logic.NewSelectCartLogic(ctx, s.svcCtx)
	return l.SelectCart(in)
}

func (s *CartServer) GetSelectedCart(ctx context.Context, in *pb.GetSelectedCartReq) (*pb.GetSelectedCartResp, error) {
	l := logic.NewGetSelectedCartLogic(ctx, s.svcCtx)
	return l.GetSelectedCart(in)
}

func (s *CartServer) Ping(ctx context.Context) error {
	logx.Info("Cart RPC Server is alive")
	return nil
}
