package service

import (
	"context"
	"errors"

	"ecommerce-demo/app/product/internal/repo"
	"ecommerce-demo/app/product/pb"
)

var ErrProductNotFound = errors.New("商品不存在")

type ProductService interface {
	GetProduct(ctx context.Context, req *pb.GetProductReq) (*pb.GetProductResp, error)
	ListProduct(ctx context.Context, req *pb.ListProductReq) (*pb.ListProductResp, error)
	AddProduct(ctx context.Context, req *pb.AddProductReq) (*pb.AddProductResp, error)
	UpdateProduct(ctx context.Context, req *pb.UpdateProductReq) (*pb.UpdateProductResp, error)
	DeleteProduct(ctx context.Context, req *pb.DeleteProductReq) (*pb.DeleteProductResp, error)
	ListProductByPage(ctx context.Context, req *pb.ListProductByPageReq) (*pb.ListProductByPageResp, error)
	GetCategories(ctx context.Context, req *pb.GetCategoriesReq) (*pb.GetCategoriesResp, error)
	AddCategory(ctx context.Context, req *pb.AddCategoryReq) (*pb.AddCategoryResp, error)
}

type productServiceImpl struct {
	repo         repo.ProductRepo
	categoryRepo repo.CategoryRepo
}

func NewProductService(repo repo.ProductRepo, categoryRepo repo.CategoryRepo) ProductService {
	return &productServiceImpl{
		repo:         repo,
		categoryRepo: categoryRepo,
	}
}

func (s *productServiceImpl) GetProduct(ctx context.Context, req *pb.GetProductReq) (*pb.GetProductResp, error) {
	p, err := s.repo.GetProductByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProductNotFound
	}

	// 获取分类名称
	categoryName := ""
	if p.CategoryID > 0 {
		if cat, _ := s.categoryRepo.GetCategoryByID(ctx, p.CategoryID); cat != nil {
			categoryName = cat.Name
		}
	}

	// 获取库存（优先从 Redis 缓存获取，缓存不存在则查数据库）
	stock, err := s.repo.GetStockFromCache(ctx, p.ID)
	if err != nil {
		// 缓存不存在，从数据库获取并写入缓存
		stock, _ = s.repo.GetStock(ctx, p.ID)
		s.repo.SetStockCache(ctx, p.ID, stock)
	}

	return &pb.GetProductResp{
		Product: &pb.ProductItem{
			Id:           p.ID,
			Name:         p.Name,
			Desc:         p.Desc,
			Price:        p.Price,
			ImageUrl:     p.ImageUrl,
			CategoryId:   p.CategoryID,
			CategoryName: categoryName,
			Stock:        stock,
		},
	}, nil
}

func (s *productServiceImpl) ListProduct(ctx context.Context, req *pb.ListProductReq) (*pb.ListProductResp, error) {
	list, err := s.repo.ListProducts(ctx)
	if err != nil {
		return nil, err
	}

	var pbList []*pb.ProductItem
	for _, p := range list {
		// 尝试从 Redis 缓存获取库存，缓存不存在则查数据库
		stock, err := s.repo.GetStockFromCache(ctx, p.ID)
		if err != nil {
			stock, _ = s.repo.GetStock(ctx, p.ID)
			s.repo.SetStockCache(ctx, p.ID, stock)
		}

		// 尝试获取分类名称
		categoryName := ""
		if p.CategoryID > 0 {
			if cat, _ := s.categoryRepo.GetCategoryByID(ctx, p.CategoryID); cat != nil {
				categoryName = cat.Name
			}
		}

		pbList = append(pbList, &pb.ProductItem{
			Id:           p.ID,
			Name:         p.Name,
			Desc:         p.Desc,
			Price:        p.Price,
			ImageUrl:     p.ImageUrl,
			CategoryId:   p.CategoryID,
			CategoryName: categoryName,
			Stock:        stock,
		})
	}

	return &pb.ListProductResp{
		Products: pbList,
	}, nil
}

func (s *productServiceImpl) AddProduct(ctx context.Context, req *pb.AddProductReq) (*pb.AddProductResp, error) {
	p := &repo.Product{
		Name:       req.Name,
		Desc:       req.Desc,
		Price:      req.Price,
		ImageUrl:   req.ImageUrl,
		CategoryID: req.CategoryId,
	}

	id, err := s.repo.AddProduct(ctx, p, req.Stock)
	if err != nil {
		return nil, err
	}

	return &pb.AddProductResp{
		Id:      id,
		Message: "商品添加成功",
	}, nil
}

func (s *productServiceImpl) UpdateProduct(ctx context.Context, req *pb.UpdateProductReq) (*pb.UpdateProductResp, error) {
	// 先检查商品是否存在
	existing, err := s.repo.GetProductByID(ctx, req.Id)
	if err != nil {
		return &pb.UpdateProductResp{
			Success: false,
			Message: "查询商品失败: " + err.Error(),
		}, nil
	}
	if existing == nil {
		return &pb.UpdateProductResp{
			Success: false,
			Message: "商品不存在",
		}, nil
	}

	// 更新商品
	p := &repo.Product{
		ID:         req.Id,
		Name:       req.Name,
		Desc:       req.Desc,
		Price:      req.Price,
		ImageUrl:   req.ImageUrl,
		CategoryID: req.CategoryId,
	}

	if err := s.repo.UpdateProduct(ctx, p); err != nil {
		return &pb.UpdateProductResp{
			Success: false,
			Message: "更新商品失败: " + err.Error(),
		}, nil
	}

	return &pb.UpdateProductResp{
		Success: true,
		Message: "商品更新成功",
	}, nil
}

func (s *productServiceImpl) DeleteProduct(ctx context.Context, req *pb.DeleteProductReq) (*pb.DeleteProductResp, error) {
	// 先检查商品是否存在
	existing, err := s.repo.GetProductByID(ctx, req.Id)
	if err != nil {
		return &pb.DeleteProductResp{
			Success: false,
			Message: "查询商品失败: " + err.Error(),
		}, nil
	}
	if existing == nil {
		return &pb.DeleteProductResp{
			Success: false,
			Message: "商品不存在",
		}, nil
	}

	// 删除商品
	if err := s.repo.DeleteProduct(ctx, req.Id); err != nil {
		return &pb.DeleteProductResp{
			Success: false,
			Message: "删除商品失败: " + err.Error(),
		}, nil
	}

	return &pb.DeleteProductResp{
		Success: true,
		Message: "商品删除成功",
	}, nil
}

func (s *productServiceImpl) ListProductByPage(ctx context.Context, req *pb.ListProductByPageReq) (*pb.ListProductByPageResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	list, total, err := s.repo.ListProductsByPage(ctx, req.CategoryId, req.Keyword, page, pageSize)
	if err != nil {
		return nil, err
	}

	// 批量获取分类和库存（减少数据库查询）
	categoryMap := make(map[int64]string)
	stockMap := make(map[int64]int32)
	var categoryIDs []int64
	var productIDs []int64

	for _, p := range list {
		if p.CategoryID > 0 {
			categoryIDs = append(categoryIDs, p.CategoryID)
		}
		productIDs = append(productIDs, p.ID)
	}

	// 批量查询分类
	categories, _ := s.categoryRepo.ListCategories(ctx)
	for _, c := range categories {
		categoryMap[c.ID] = c.Name
	}

	// 批量查询库存（直接从 Redis 获取，不查 MySQL）
	for _, pid := range productIDs {
		if stock, err := s.repo.GetStockFromCache(ctx, pid); err == nil {
			stockMap[pid] = stock
		} else {
			// 降级：从数据库查
			stock, _ = s.repo.GetStock(ctx, pid)
			stockMap[pid] = stock
		}
	}

	var pbList []*pb.ProductItem
	for _, p := range list {
		pbList = append(pbList, &pb.ProductItem{
			Id:           p.ID,
			Name:         p.Name,
			Desc:         p.Desc,
			Price:        p.Price,
			ImageUrl:     p.ImageUrl,
			CategoryId:   p.CategoryID,
			CategoryName: categoryMap[p.CategoryID],
			Stock:        stockMap[p.ID],
		})
	}

	return &pb.ListProductByPageResp{
		Products: pbList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *productServiceImpl) GetCategories(ctx context.Context, req *pb.GetCategoriesReq) (*pb.GetCategoriesResp, error) {
	list, err := s.categoryRepo.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	var pbList []*pb.Category
	for _, c := range list {
		pbList = append(pbList, &pb.Category{
			Id:   c.ID,
			Name: c.Name,
			Icon: c.Icon,
			Sort: c.Sort,
		})
	}

	return &pb.GetCategoriesResp{
		Categories: pbList,
	}, nil
}

func (s *productServiceImpl) AddCategory(ctx context.Context, req *pb.AddCategoryReq) (*pb.AddCategoryResp, error) {
	c := &repo.Category{
		Name: req.Name,
		Icon: req.Icon,
		Sort: req.Sort,
	}

	id, err := s.categoryRepo.AddCategory(ctx, c)
	if err != nil {
		return nil, err
	}

	return &pb.AddCategoryResp{
		Id:      id,
		Message: "分类添加成功",
	}, nil
}
