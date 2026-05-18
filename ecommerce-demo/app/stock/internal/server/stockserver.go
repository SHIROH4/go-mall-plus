package server

import (
	"context"

	"ecommerce-demo/app/stock/internal/svc"
	"ecommerce-demo/app/stock/pb"
)

type StockServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedStockServer
}

func NewStockServer(svcCtx *svc.ServiceContext) *StockServer {
	return &StockServer{svcCtx: svcCtx}
}

func (s *StockServer) DeductStock(ctx context.Context, in *pb.DeductStockReq) (*pb.DeductStockResp, error) {
	return s.svcCtx.StockService.DeductStock(ctx, in)
}

func (s *StockServer) RollbackStock(ctx context.Context, in *pb.RollbackStockReq) (*pb.RollbackStockResp, error) {
	return s.svcCtx.StockService.RollbackStock(ctx, in)
}

func (s *StockServer) GetStock(ctx context.Context, in *pb.GetStockReq) (*pb.GetStockResp, error) {
	return s.svcCtx.StockService.GetStock(ctx, in)
}

func (s *StockServer) BatchGetStock(ctx context.Context, in *pb.BatchGetStockReq) (*pb.BatchGetStockResp, error) {
	return s.svcCtx.StockService.BatchGetStock(ctx, in)
}
