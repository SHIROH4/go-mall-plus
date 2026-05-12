package svc

import (
	"ecommerce-demo/app/address/internal/config"
	"ecommerce-demo/app/address/internal/repo"
	"ecommerce-demo/app/address/internal/repo/mysqlrepo"
	"ecommerce-demo/app/address/internal/service"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config         config.Config
	AddressRepo    repo.AddressRepo
	AddressService service.AddressService
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(mysql.Open(c.DataSource), &gorm.Config{})
	if err != nil {
		panic("连接数据库失败: " + err.Error())
	}

	addressRepo := mysqlrepo.NewAddressRepo(db)
	addressService := service.NewAddressService(addressRepo)

	return &ServiceContext{
		Config:         c,
		AddressRepo:    addressRepo,
		AddressService: addressService,
	}
}
