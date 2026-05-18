package service

import (
	"context"
	"errors"

	"ecommerce-demo/app/stock/internal/repo"
	"ecommerce-demo/app/stock/pb"
	"ecommerce-demo/common/metrics"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrStockNotEnough    = errors.New("库存不足")
	ErrStockNotInitiated = errors.New("库存缓存未初始化")
)

type StockService interface {
	DeductStock(ctx context.Context, req *pb.DeductStockReq) (*pb.DeductStockResp, error)
	RollbackStock(ctx context.Context, req *pb.RollbackStockReq) (*pb.RollbackStockResp, error)
	GetStock(ctx context.Context, req *pb.GetStockReq) (*pb.GetStockResp, error)
	BatchGetStock(ctx context.Context, req *pb.BatchGetStockReq) (*pb.BatchGetStockResp, error)
}

type stockServiceImpl struct {
	repo repo.StockRepo
	logx.Logger
}

func NewStockService(repo repo.StockRepo) StockService {
	return &stockServiceImpl{
		repo:   repo,
		Logger: logx.WithContext(context.Background()),
	}
}

// DeductStock 扣减库存（核心秒杀路径，仅依赖 Redis Lua）
func (s *stockServiceImpl) DeductStock(ctx context.Context, req *pb.DeductStockReq) (*pb.DeductStockResp, error) {
	if req.ProductId <= 0 || req.Count <= 0 {
		return &pb.DeductStockResp{Success: false, Message: "参数非法"}, nil
	}

	ok, err := s.repo.DeductStockLua(ctx, req.ProductId, req.Count)
	if err != nil {
		metrics.StockDeductTotal.WithLabelValues("fail").Inc()
		if errors.Is(err, repo.ErrStockNotInitiated) {
			return &pb.DeductStockResp{Success: false, Message: "库存缓存未初始化，请稍后再试"}, nil
		}
		s.Errorf("扣减库存失败: productID=%d, err=%v", req.ProductId, err)
		return &pb.DeductStockResp{Success: false, Message: "系统繁忙，请稍后再试"}, nil
	}

	if !ok {
		metrics.StockDeductTotal.WithLabelValues("fail").Inc()
		return &pb.DeductStockResp{Success: false, Message: "手慢了，商品库存不足"}, nil
	}

	metrics.StockDeductTotal.WithLabelValues("success").Inc()

	s.Infof("库存扣减成功: productID=%d, count=%d", req.ProductId, req.Count)
	return &pb.DeductStockResp{Success: true, Message: "ok"}, nil
}

// RollbackStock 回滚库存
func (s *stockServiceImpl) RollbackStock(ctx context.Context, req *pb.RollbackStockReq) (*pb.RollbackStockResp, error) {
	if err := s.repo.RollbackStockLua(ctx, req.ProductId, req.Count); err != nil {
		s.Errorf("回滚库存失败: productID=%d, err=%v", req.ProductId, err)
		return &pb.RollbackStockResp{Success: false}, nil
	}
	s.Infof("库存回滚成功: productID=%d, count=%d", req.ProductId, req.Count)
	return &pb.RollbackStockResp{Success: true}, nil
}

// GetStock 查询库存
func (s *stockServiceImpl) GetStock(ctx context.Context, req *pb.GetStockReq) (*pb.GetStockResp, error) {
	stock, err := s.repo.GetStockFromCache(ctx, req.ProductId)
	if err != nil {
		s.Errorf("查询库存失败: productID=%d, err=%v", req.ProductId, err)
		// 降级查 MySQL
		stock, _ = s.repo.GetStock(ctx, req.ProductId)
	}
	return &pb.GetStockResp{Stock: stock}, nil
}

// BatchGetStock 批量查询库存
func (s *stockServiceImpl) BatchGetStock(ctx context.Context, req *pb.BatchGetStockReq) (*pb.BatchGetStockResp, error) {
	stocks, err := s.repo.BatchGetStockFromCache(ctx, req.ProductIds)
	if err != nil {
		s.Errorf("批量查询库存失败: err=%v", err)
		return &pb.BatchGetStockResp{Stocks: map[int64]int32{}}, nil
	}
	return &pb.BatchGetStockResp{Stocks: stocks}, nil
}
