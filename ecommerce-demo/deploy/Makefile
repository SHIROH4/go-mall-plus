.PHONY: help build push deploy uninstall status logs

# 配置区域
REGISTRY ?= your-registry.com
IMAGE_TAG ?= latest
NAMESPACE ?= ecommerce
RELEASE_NAME ?= ecommerce

# 路径
SCRIPTS_DIR := $(shell realpath $(dir $(lastword $(MAKEFILE_LIST)))/scripts)
PROJECT_DIR := $(shell realpath $(dir $(SCRIPTS_DIR)))

help: ## 显示帮助
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =============================================================================
# 构建相关
# =============================================================================

build: ## 构建所有服务镜像
	@bash $(SCRIPTS_DIR)/build.sh build

push: ## 推送所有镜像
	@bash $(SCRIPTS_DIR)/build.sh push

build-push: build push ## 构建并推送所有镜像

build-gateway: ## 构建 gateway
	docker build -t $(REGISTRY)/ecommerce-gateway:$(IMAGE_TAG) -f $(PROJECT_DIR)/app/gateway/Dockerfile $(PROJECT_DIR)

build-user: ## 构建 user
	docker build -t $(REGISTRY)/ecommerce-user:$(IMAGE_TAG) -f $(PROJECT_DIR)/app/user/Dockerfile $(PROJECT_DIR)

build-product: ## 构建 product
	docker build -t $(REGISTRY)/ecommerce-product:$(IMAGE_TAG) -f $(PROJECT_DIR)/app/product/Dockerfile $(PROJECT_DIR)

build-cart: ## 构建 cart
	docker build -t $(REGISTRY)/ecommerce-cart:$(IMAGE_TAG) -f $(PROJECT_DIR)/app/cart/Dockerfile $(PROJECT_DIR)

build-order: ## 构建 order
	docker build -t $(REGISTRY)/ecommerce-order:$(IMAGE_TAG) -f $(PROJECT_DIR)/app/order/Dockerfile $(PROJECT_DIR)

build-payment: ## 构建 payment
	docker build -t $(REGISTRY)/ecommerce-payment:$(IMAGE_TAG) -f $(PROJECT_DIR)/app/payment/Dockerfile $(PROJECT_DIR)

build-address: ## 构建 address
	docker build -t $(REGISTRY)/ecommerce-address:$(IMAGE_TAG) -f $(PROJECT_DIR)/app/address/Dockerfile $(PROJECT_DIR)

# =============================================================================
# Helm 部署相关
# =============================================================================

deploy: ## 部署到 K8s
	@bash $(SCRIPTS_DIR)/deploy.sh deploy

uninstall: ## 卸载
	@bash $(SCRIPTS_DIR)/deploy.sh uninstall

status: ## 查看状态
	@bash $(SCRIPTS_DIR)/deploy.sh status

logs: ## 查看日志 (SVC=gateway)
	@bash $(SCRIPTS_DIR)/deploy.sh logs $(SVC)

port-forward: ## 端口转发 (PORT=8888)
	@bash $(SCRIPTS_DIR)/deploy.sh port-forward gateway $(PORT)

# =============================================================================
# 单机 Docker Compose 部署
# =============================================================================

run-build-deps: ## 预构建依赖 (首次运行，加速后续构建)
	@bash $(SCRIPTS_DIR)/run.sh build-deps

run-build-all: ## 预构建所有服务镜像
	@bash $(SCRIPTS_DIR)/run.sh build-all

run-infra: ## 启动基础设施 (MySQL/Redis/Etcd/RabbitMQ)
	@bash $(SCRIPTS_DIR)/run.sh infra-start

run-stop-infra: ## 停止基础设施
	@bash $(SCRIPTS_DIR)/run.sh infra-stop

run-start: ## 启动单个服务 (SVC=gateway)
	@bash $(SCRIPTS_DIR)/run.sh start $(SVC)

run-stop: ## 停止单个服务 (SVC=gateway)
	@bash $(SCRIPTS_DIR)/run.sh stop $(SVC)

run-gateway: ## 启动 Gateway
	@bash $(SCRIPTS_DIR)/run.sh gateway

run-status: ## 查看所有服务状态
	@bash $(SCRIPTS_DIR)/run.sh status

run-logs: ## 查看日志 (SVC=gateway)
	@bash $(SCRIPTS_DIR)/run.sh logs $(SVC)

run-clean: ## 清理所有
	@bash $(SCRIPTS_DIR)/run.sh clean

# =============================================================================
# Kind 本地测试
# =============================================================================

kind-create: ## 创建 Kind 集群
	@bash $(SCRIPTS_DIR)/kind.sh create

kind-deploy: ## 部署到 Kind
	@bash $(SCRIPTS_DIR)/kind.sh deploy

kind-all: ## 一键创建 Kind 并部署
	@bash $(SCRIPTS_DIR)/kind.sh all

kind-cleanup: ## 删除 Kind 集群
	@bash $(SCRIPTS_DIR)/kind.sh cleanup

# =============================================================================
# 开发相关
# =============================================================================

dev-deploy: ## 开发环境部署（1副本）
	helm upgrade --install $(RELEASE_NAME) deploy/helm/ecommerce \
		--namespace $(NAMESPACE) \
		--create-namespace \
		--set global.registry=$(REGISTRY) \
		--set gateway.replicaCount=1 \
		--set services.user.replicaCount=1 \
		--set services.product.replicaCount=1 \
		--set services.cart.replicaCount=1 \
		--set services.order.replicaCount=1 \
		--set services.payment.replicaCount=1 \
		--set services.address.replicaCount=1 \
		--wait --timeout 5m

restart: ## 重启所有 Pod
	kubectl delete pods --all -n $(NAMESPACE)

# =============================================================================
# 清理
# =============================================================================

clean-images: ## 清理本地镜像
	@docker rmi $$(docker images | grep ecommerce | awk '{print $3}') 2>/dev/null || true

prune: ## 清理未使用的资源
	kubectl delete pods --all -n $(NAMESPACE) --grace-period=0 --force || true
	kubectl delete pvc --all -n $(NAMESPACE) || true
