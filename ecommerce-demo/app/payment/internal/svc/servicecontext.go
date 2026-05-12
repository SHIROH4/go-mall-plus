package svc

import (
	"ecommerce-demo/app/payment/internal/config"
	"ecommerce-demo/app/payment/internal/model"
	"ecommerce-demo/app/payment/internal/repo/mysql"
	"ecommerce-demo/app/payment/internal/service"
	orderclient "ecommerce-demo/app/order/order"

	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config         config.Config
	PaymentService service.PaymentService
}

func NewServiceContext(c config.Config, orderRpc orderclient.Order) *ServiceContext {
	// 初始化 MySQL
	db, err := gorm.Open(gormMysql.Open(c.MySQLConf.DataSource), &gorm.Config{})
	if err != nil {
		panic("连接 MySQL 失败: " + err.Error())
	}

	// 自动迁移表
	db.AutoMigrate(&model.Payment{})

	// 初始化 Repo
	paymentRepo := mysql.NewPaymentRepo(db)

	// 初始化 Service
	paymentService := service.NewPaymentServiceWithConfig(paymentRepo, orderRpc, c.PayExpireMinutes)

	return &ServiceContext{
		Config:         c,
		PaymentService: paymentService,
	}
}
