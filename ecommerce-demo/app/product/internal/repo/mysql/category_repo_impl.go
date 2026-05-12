package mysql

import (
	"context"

	"ecommerce-demo/app/product/internal/repo"

	"gorm.io/gorm"
)

type categoryRepoImpl struct {
	db *gorm.DB
}

func NewCategoryRepo(db *gorm.DB) repo.CategoryRepo {
	return &categoryRepoImpl{db: db}
}

func (r *categoryRepoImpl) ListCategories(ctx context.Context) ([]*repo.Category, error) {
	var categories []*repo.Category
	err := r.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *categoryRepoImpl) AddCategory(ctx context.Context, c *repo.Category) (int64, error) {
	err := r.db.WithContext(ctx).Create(c).Error
	if err != nil {
		return 0, err
	}
	return c.ID, nil
}

func (r *categoryRepoImpl) GetCategoryByID(ctx context.Context, id int64) (*repo.Category, error) {
	var category repo.Category
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}
