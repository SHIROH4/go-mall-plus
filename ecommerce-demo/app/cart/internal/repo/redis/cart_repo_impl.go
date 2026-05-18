package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ecommerce-demo/app/cart/internal/repo"
	"ecommerce-demo/app/cart/pb"

	"github.com/redis/go-redis/v9"
)

type cartRedisRepo struct {
	rdb *redis.ClusterClient
}

func NewCartRedisRepo(rdb *redis.ClusterClient) repo.CartRepo {
	return &cartRedisRepo{rdb: rdb}
}

func (r *cartRedisRepo) cartKey(userId int64) string {
	return fmt.Sprintf("%s%d", repo.CartKeyPrefix, userId)
}

func (r *cartRedisRepo) itemKey(productId int64) string {
	return fmt.Sprintf("%d", productId)
}

func (r *cartRedisRepo) AddItem(ctx context.Context, userId int64, item *pb.CartItem) error {
	key := r.cartKey(userId)
	field := r.itemKey(item.ProductId)

	// 先检查商品是否已存在
	existing, err := r.rdb.HGet(ctx, key, field).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	now := time.Now().Unix()
	if err == redis.Nil {
		// 商品不存在，检查购物车是否已满
		size, err := r.rdb.HLen(ctx, key).Result()
		if err != nil {
			return err
		}
		if size >= repo.MaxCartSize {
			return repo.ErrCartFull
		}

		// 新增商品
		item.CreatedAt = now
		item.UpdatedAt = now
	} else {
		// 商品已存在，累加数量
		var existingItem pb.CartItem
		if err := json.Unmarshal([]byte(existing), &existingItem); err != nil {
			return err
		}

		newCount := existingItem.Count + item.Count
		if newCount > repo.MaxItemCount {
			return repo.ErrItemCountLimit
		}

		item.Count = newCount
		item.CreatedAt = existingItem.CreatedAt
		item.UpdatedAt = now
	}

	// 序列化并写入
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	pipe := r.rdb.Pipeline()
	pipe.HSet(ctx, key, field, data)
	pipe.Expire(ctx, key, repo.CartExpire)

	_, err = pipe.Exec(ctx)
	return err
}

func (r *cartRedisRepo) GetCart(ctx context.Context, userId int64) ([]*pb.CartItem, error) {
	key := r.cartKey(userId)

	result, err := r.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return []*pb.CartItem{}, nil
	}

	items := make([]*pb.CartItem, 0, len(result))
	for _, data := range result {
		var item pb.CartItem
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			continue
		}
		items = append(items, &item)
	}

	return items, nil
}

func (r *cartRedisRepo) UpdateItem(ctx context.Context, userId int64, productId int64, count int32) error {
	key := r.cartKey(userId)
	field := r.itemKey(productId)

	// 获取现有商品
	data, err := r.rdb.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return repo.ErrItemNotFound
	}
	if err != nil {
		return err
	}

	var item pb.CartItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return err
	}

	// 数量为 0 时删除商品
	if count == 0 {
		return r.RemoveItem(ctx, userId, productId)
	}

	if count > repo.MaxItemCount {
		return repo.ErrItemCountLimit
	}

	item.Count = count
	item.UpdatedAt = time.Now().Unix()

	newData, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return r.rdb.HSet(ctx, key, field, newData).Err()
}

func (r *cartRedisRepo) RemoveItem(ctx context.Context, userId int64, productId int64) error {
	key := r.cartKey(userId)
	field := r.itemKey(productId)

	result, err := r.rdb.HDel(ctx, key, field).Result()
	if err != nil {
		return err
	}
	if result == 0 {
		return repo.ErrItemNotFound
	}
	return nil
}

func (r *cartRedisRepo) ClearCart(ctx context.Context, userId int64) (int32, error) {
	key := r.cartKey(userId)

	size, err := r.rdb.HLen(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if err := r.rdb.Del(ctx, key).Err(); err != nil {
		return 0, err
	}

	return int32(size), nil
}

func (r *cartRedisRepo) SelectItem(ctx context.Context, userId int64, productId int64, selected bool) error {
	key := r.cartKey(userId)
	field := r.itemKey(productId)

	data, err := r.rdb.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return repo.ErrItemNotFound
	}
	if err != nil {
		return err
	}

	var item pb.CartItem
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return err
	}

	item.Selected = selected
	item.UpdatedAt = time.Now().Unix()

	newData, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return r.rdb.HSet(ctx, key, field, newData).Err()
}

func (r *cartRedisRepo) GetSelectedItems(ctx context.Context, userId int64) ([]*pb.CartItem, error) {
	items, err := r.GetCart(ctx, userId)
	if err != nil {
		return nil, err
	}

	selectedItems := make([]*pb.CartItem, 0)
	for _, item := range items {
		if item.Selected {
			selectedItems = append(selectedItems, item)
		}
	}

	return selectedItems, nil
}

func (r *cartRedisRepo) GetCartSize(ctx context.Context, userId int64) (int32, error) {
	key := r.cartKey(userId)
	size, err := r.rdb.HLen(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int32(size), nil
}

func (r *cartRedisRepo) CartKey(userId int64) string {
	return r.cartKey(userId)
}

func (r *cartRedisRepo) ItemKey(productId int64) string {
	return r.itemKey(productId)
}
