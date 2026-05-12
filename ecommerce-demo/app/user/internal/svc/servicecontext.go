package svc

import (
	"log"

	"ecommerce-demo/app/user/internal/config"
	"ecommerce-demo/app/user/internal/repo"
	"ecommerce-demo/app/user/internal/repo/mysql"
	"ecommerce-demo/app/user/internal/service"

	gormMysql "gorm.io/driver/mysql" // 给原始驱动起别名，防冲突
	"gorm.io/gorm"
)

// ServiceContext 依赖注入容器：这里挂载系统级别的依赖，如 DB、Redis、Repo 等
type ServiceContext struct {
	Config      config.Config
	UserRepo    repo.UserRepo       // 仓储接口
	UserService service.UserService // 业务接口
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 1. 初始化 GORM
	db, err := gorm.Open(gormMysql.Open(c.DataSource), &gorm.Config{})
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	// 2. 初始化 Repo 实现类
	userRepo := mysql.NewUserRepo(db)

	// 3、 初始化 Service（将Repo注入到Service）
	userService := service.NewUserService(userRepo)

	return &ServiceContext{
		Config:      c,
		UserRepo:    userRepo,
		UserService: userService,
	}
}
