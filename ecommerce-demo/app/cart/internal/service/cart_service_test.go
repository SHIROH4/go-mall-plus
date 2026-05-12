package service

import (
	"context"
	"testing"

	"ecommerce-demo/app/cart/internal/repo"
	"ecommerce-demo/app/cart/internal/service"
	"ecommerce-demo/app/cart/pb"
)

type mockCartRepo struct {
	data map[int64]map[int64]*pb.CartItem
}

func newMockCartRepo() *mockCartRepo {
	return &mockCartRepo{
		data: make(map[int64]map[int64]*pb.CartItem),
	}
}

func (m *mockCartRepo) AddItem(ctx context.Context, userId int64, item *pb.CartItem) error {
	if m.data[userId] == nil {
		m.data[userId] = make(map[int64]*pb.CartItem)
	}

	if existing, ok := m.data[userId][item.ProductId]; ok {
		existing.Count += item.Count
		existing.UpdatedAt = item.UpdatedAt
	} else {
		if len(m.data[userId]) >= repo.MaxCartSize {
			return repo.ErrCartFull
		}
		m.data[userId][item.ProductId] = item
	}
	return nil
}

func (m *mockCartRepo) GetCart(ctx context.Context, userId int64) ([]*pb.CartItem, error) {
	if m.data[userId] == nil {
		return []*pb.CartItem{}, nil
	}

	items := make([]*pb.CartItem, 0)
	for _, item := range m.data[userId] {
		items = append(items, item)
	}
	return items, nil
}

func (m *mockCartRepo) UpdateItem(ctx context.Context, userId int64, productId int64, count int32) error {
	if m.data[userId] == nil || m.data[userId][productId] == nil {
		return repo.ErrItemNotFound
	}
	if count > repo.MaxItemCount {
		return repo.ErrItemCountLimit
	}
	m.data[userId][productId].Count = count
	return nil
}

func (m *mockCartRepo) RemoveItem(ctx context.Context, userId int64, productId int64) error {
	if m.data[userId] == nil || m.data[userId][productId] == nil {
		return repo.ErrItemNotFound
	}
	delete(m.data[userId], productId)
	return nil
}

func (m *mockCartRepo) ClearCart(ctx context.Context, userId int64) (int32, error) {
	if m.data[userId] == nil {
		return 0, nil
	}
	count := int32(len(m.data[userId]))
	delete(m.data, userId)
	return count, nil
}

func (m *mockCartRepo) SelectItem(ctx context.Context, userId int64, productId int64, selected bool) error {
	if m.data[userId] == nil || m.data[userId][productId] == nil {
		return repo.ErrItemNotFound
	}
	m.data[userId][productId].Selected = selected
	return nil
}

func (m *mockCartRepo) GetSelectedItems(ctx context.Context, userId int64) ([]*pb.CartItem, error) {
	items, _ := m.GetCart(ctx, userId)
	selected := make([]*pb.CartItem, 0)
	for _, item := range items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return selected, nil
}

func (m *mockCartRepo) GetCartSize(ctx context.Context, userId int64) (int32, error) {
	if m.data[userId] == nil {
		return 0, nil
	}
	return int32(len(m.data[userId])), nil
}

func (m *mockCartRepo) CartKey(userId int64) string {
	return "cart:" + string(rune(userId))
}

func (m *mockCartRepo) ItemKey(productId int64) string {
	return "product:" + string(rune(productId))
}

func TestAddCart(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()
	req := &pb.AddCartReq{
		UserId:      1001,
		ProductId:   2001,
		ProductName: "iPhone 15",
		Price:       599900,
		ImageUrl:    "https://example.com/iphone15.jpg",
		Count:       2,
	}

	resp, err := svc.AddCart(ctx, req)
	if err != nil {
		t.Fatalf("AddCart failed: %v", err)
	}

	if resp.Message != "添加成功" {
		t.Errorf("Expected message '添加成功', got '%s'", resp.Message)
	}

	if resp.TotalCount != 1 {
		t.Errorf("Expected total count 1, got %d", resp.TotalCount)
	}
}

func TestAddCart_ItemExists(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 第一次添加
	req1 := &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2001,
		Count:     2,
	}
	svc.AddCart(ctx, req1)

	// 第二次添加同一商品
	req2 := &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2001,
		Count:     3,
	}
	resp, err := svc.AddCart(ctx, req2)
	if err != nil {
		t.Fatalf("AddCart failed: %v", err)
	}

	// 验证数量累加
	cart, _ := svc.GetCart(ctx, &pb.GetCartReq{UserId: 1001})
	if len(cart.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(cart.Items))
	}
	if cart.Items[0].Count != 5 {
		t.Errorf("Expected count 5, got %d", cart.Items[0].Count)
	}

	if resp.TotalCount != 1 {
		t.Errorf("Expected total count 1, got %d", resp.TotalCount)
	}
}

func TestGetCart(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 添加商品
	svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2001,
		Price:    100,
		Count:    2,
	})
	svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2002,
		Price:    200,
		Count:    1,
	})

	// 获取购物车
	resp, err := svc.GetCart(ctx, &pb.GetCartReq{UserId: 1001})
	if err != nil {
		t.Fatalf("GetCart failed: %v", err)
	}

	if resp.TotalCount != 2 {
		t.Errorf("Expected 2 items, got %d", resp.TotalCount)
	}

	expectedAmount := int64(400) // 100*2 + 200*1
	if resp.TotalAmount != expectedAmount {
		t.Errorf("Expected total amount %d, got %d", expectedAmount, resp.TotalAmount)
	}
}

func TestUpdateCart(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 添加商品
	svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2001,
		Count:     5,
	})

	// 更新数量
	_, err := svc.UpdateCart(ctx, &pb.UpdateCartReq{
		UserId:    1001,
		ProductId: 2001,
		Count:     3,
	})
	if err != nil {
		t.Fatalf("UpdateCart failed: %v", err)
	}

	cart, _ := svc.GetCart(ctx, &pb.GetCartReq{UserId: 1001})
	if cart.Items[0].Count != 3 {
		t.Errorf("Expected count 3, got %d", cart.Items[0].Count)
	}
}

func TestRemoveCart(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 添加商品
	svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2001,
		Count:     1,
	})

	// 删除商品
	_, err := svc.RemoveCart(ctx, &pb.RemoveCartReq{
		UserId:    1001,
		ProductId: 2001,
	})
	if err != nil {
		t.Fatalf("RemoveCart failed: %v", err)
	}

	cart, _ := svc.GetCart(ctx, &pb.GetCartReq{UserId: 1001})
	if len(cart.Items) != 0 {
		t.Errorf("Expected 0 items after removal, got %d", len(cart.Items))
	}
}

func TestClearCart(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 添加多个商品
	for i := int64(2001); i <= 2003; i++ {
		svc.AddCart(ctx, &pb.AddCartReq{
			UserId:    1001,
			ProductId: i,
			Count:     1,
		})
	}

	// 清空购物车
	resp, err := svc.ClearCart(ctx, &pb.ClearCartReq{UserId: 1001})
	if err != nil {
		t.Fatalf("ClearCart failed: %v", err)
	}

	if resp.RemovedCount != 3 {
		t.Errorf("Expected removed count 3, got %d", resp.RemovedCount)
	}

	cart, _ := svc.GetCart(ctx, &pb.GetCartReq{UserId: 1001})
	if len(cart.Items) != 0 {
		t.Errorf("Expected 0 items after clear, got %d", len(cart.Items))
	}
}

func TestSelectCart(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 添加商品（默认选中）
	svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2001,
		Count:     1,
	})

	// 取消勾选
	_, err := svc.SelectCart(ctx, &pb.SelectCartReq{
		UserId:    1001,
		ProductId: 2001,
		Selected:  false,
	})
	if err != nil {
		t.Fatalf("SelectCart failed: %v", err)
	}

	// 获取已选中商品
	selected, _ := svc.GetSelectedCart(ctx, &pb.GetSelectedCartReq{UserId: 1001})
	if len(selected.Items) != 0 {
		t.Errorf("Expected 0 selected items, got %d", len(selected.Items))
	}
}

func TestGetSelectedCart(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 添加商品（默认选中）
	svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2001,
		Price:    100,
		Count:     2,
	})
	svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2002,
		Price:    200,
		Count:     1,
	})

	// 取消勾选一个
	svc.SelectCart(ctx, &pb.SelectCartReq{
		UserId:    1001,
		ProductId: 2002,
		Selected:  false,
	})

	// 获取已选中商品
	resp, err := svc.GetSelectedCart(ctx, &pb.GetSelectedCartReq{UserId: 1001})
	if err != nil {
		t.Fatalf("GetSelectedCart failed: %v", err)
	}

	if resp.SelectedCount != 1 {
		t.Errorf("Expected 1 selected item, got %d", resp.SelectedCount)
	}

	expectedAmount := int64(200) // 100*2
	if resp.TotalAmount != expectedAmount {
		t.Errorf("Expected total amount %d, got %d", expectedAmount, resp.TotalAmount)
	}
}

func TestCartFull(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 添加最大数量的商品
	for i := int64(1); i <= repo.MaxCartSize; i++ {
		_, err := svc.AddCart(ctx, &pb.AddCartReq{
			UserId:    1001,
			ProductId: 3000 + i,
			Count:     1,
		})
		if err != nil {
			t.Fatalf("Failed to add item %d: %v", i, err)
		}
	}

	// 再添加一个商品应该失败
	_, err := svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 4000,
		Count:     1,
	})
	if err != service.ErrCartFull {
		t.Errorf("Expected ErrCartFull, got %v", err)
	}
}

func TestUpdateCountToZero(t *testing.T) {
	repo := newMockCartRepo()
	svc := service.NewCartService(repo)

	ctx := context.Background()

	// 添加商品
	svc.AddCart(ctx, &pb.AddCartReq{
		UserId:    1001,
		ProductId: 2001,
		Count:     5,
	})

	// 更新数量为 0，应该删除商品
	resp, err := svc.UpdateCart(ctx, &pb.UpdateCartReq{
		UserId:    1001,
		ProductId: 2001,
		Count:     0,
	})
	if err != nil {
		t.Fatalf("UpdateCart failed: %v", err)
	}

	if resp.Message != "商品已从购物车移除" {
		t.Errorf("Expected message '商品已从购物车移除', got '%s'", resp.Message)
	}

	cart, _ := svc.GetCart(ctx, &pb.GetCartReq{UserId: 1001})
	if len(cart.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(cart.Items))
	}
}
