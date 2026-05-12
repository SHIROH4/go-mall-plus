package service

import (
	"context"
	"errors"

	"ecommerce-demo/app/cart/internal/repo"
	"ecommerce-demo/app/cart/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrCartEmpty      = errors.New("购物车为空")
	ErrItemNotFound   = errors.New("商品不在购物车中")
	ErrItemCountLimit = errors.New("单商品数量已达上限 (最多99件)")
	ErrCartFull       = errors.New("购物车已满 (最多50种商品)")
)

type CartService interface {
	AddCart(ctx context.Context, req *pb.AddCartReq) (*pb.AddCartResp, error)
	GetCart(ctx context.Context, req *pb.GetCartReq) (*pb.GetCartResp, error)
	UpdateCart(ctx context.Context, req *pb.UpdateCartReq) (*pb.UpdateCartResp, error)
	RemoveCart(ctx context.Context, req *pb.RemoveCartReq) (*pb.RemoveCartResp, error)
	ClearCart(ctx context.Context, req *pb.ClearCartReq) (*pb.ClearCartResp, error)
	SelectCart(ctx context.Context, req *pb.SelectCartReq) (*pb.SelectCartResp, error)
	GetSelectedCart(ctx context.Context, req *pb.GetSelectedCartReq) (*pb.GetSelectedCartResp, error)
}

type cartServiceImpl struct {
	repo repo.CartRepo
	logx.Logger
}

func NewCartService(repo repo.CartRepo) CartService {
	return &cartServiceImpl{
		repo:   repo,
		Logger: logx.WithContext(context.Background()),
	}
}

func (s *cartServiceImpl) AddCart(ctx context.Context, req *pb.AddCartReq) (*pb.AddCartResp, error) {
	item := &pb.CartItem{
		ProductId:   req.ProductId,
		ProductName: req.ProductName,
		Price:       req.Price,
		ImageUrl:    req.ImageUrl,
		Count:       req.Count,
		Selected:    true,
	}

	if err := s.repo.AddItem(ctx, req.UserId, item); err != nil {
		if errors.Is(err, repo.ErrCartFull) {
			return nil, ErrCartFull
		}
		if errors.Is(err, repo.ErrItemCountLimit) {
			return nil, ErrItemCountLimit
		}
		return nil, err
	}

	size, _ := s.repo.GetCartSize(ctx, req.UserId)
	return &pb.AddCartResp{
		Message:    "添加成功",
		TotalCount: size,
	}, nil
}

func (s *cartServiceImpl) GetCart(ctx context.Context, req *pb.GetCartReq) (*pb.GetCartResp, error) {
	items, err := s.repo.GetCart(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	var totalAmount int64
	for _, item := range items {
		totalAmount += item.Price * int64(item.Count)
	}

	return &pb.GetCartResp{
		Items:       items,
		TotalCount:  int32(len(items)),
		TotalAmount: totalAmount,
	}, nil
}

func (s *cartServiceImpl) UpdateCart(ctx context.Context, req *pb.UpdateCartReq) (*pb.UpdateCartResp, error) {
	if req.Count == 0 {
		if err := s.repo.RemoveItem(ctx, req.UserId, req.ProductId); err != nil {
			if errors.Is(err, repo.ErrItemNotFound) {
				return nil, ErrItemNotFound
			}
			return nil, err
		}
		return &pb.UpdateCartResp{Message: "商品已从购物车移除"}, nil
	}

	if err := s.repo.UpdateItem(ctx, req.UserId, req.ProductId, req.Count); err != nil {
		if errors.Is(err, repo.ErrItemNotFound) {
			return nil, ErrItemNotFound
		}
		if errors.Is(err, repo.ErrItemCountLimit) {
			return nil, ErrItemCountLimit
		}
		return nil, err
	}

	return &pb.UpdateCartResp{Message: "更新成功"}, nil
}

func (s *cartServiceImpl) RemoveCart(ctx context.Context, req *pb.RemoveCartReq) (*pb.RemoveCartResp, error) {
	if err := s.repo.RemoveItem(ctx, req.UserId, req.ProductId); err != nil {
		if errors.Is(err, repo.ErrItemNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	return &pb.RemoveCartResp{Message: "删除成功"}, nil
}

func (s *cartServiceImpl) ClearCart(ctx context.Context, req *pb.ClearCartReq) (*pb.ClearCartResp, error) {
	removedCount, err := s.repo.ClearCart(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &pb.ClearCartResp{
		Message:      "清空成功",
		RemovedCount: removedCount,
	}, nil
}

func (s *cartServiceImpl) SelectCart(ctx context.Context, req *pb.SelectCartReq) (*pb.SelectCartResp, error) {
	if err := s.repo.SelectItem(ctx, req.UserId, req.ProductId, req.Selected); err != nil {
		if errors.Is(err, repo.ErrItemNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	action := "已勾选"
	if !req.Selected {
		action = "已取消勾选"
	}
	return &pb.SelectCartResp{Message: action}, nil
}

func (s *cartServiceImpl) GetSelectedCart(ctx context.Context, req *pb.GetSelectedCartReq) (*pb.GetSelectedCartResp, error) {
	items, err := s.repo.GetSelectedItems(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return &pb.GetSelectedCartResp{
			Items:          []*pb.CartItem{},
			SelectedCount:  0,
			TotalAmount:    0,
		}, nil
	}

	var totalAmount int64
	for _, item := range items {
		totalAmount += item.Price * int64(item.Count)
	}

	return &pb.GetSelectedCartResp{
		Items:         items,
		SelectedCount: int32(len(items)),
		TotalAmount:   totalAmount,
	}, nil
}
