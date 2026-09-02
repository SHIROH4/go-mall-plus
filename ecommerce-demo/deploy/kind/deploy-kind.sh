#!/bin/bash
# =============================================================================
# Kind Cluster Setup and Deployment Script (K8s native - no Etcd)
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIND_DIR="$SCRIPT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-ecommerce-cluster}"

# =============================================================================
# Check prerequisites
# =============================================================================
check_prerequisites() {
    log_info "Checking prerequisites..."

    command -v kind >/dev/null 2>&1 || { log_error "Kind is not installed."; exit 1; }
    command -v kubectl >/dev/null 2>&1 || { log_error "kubectl is not installed."; exit 1; }
    command -v docker >/dev/null 2>&1 || { log_error "Docker is not installed."; exit 1; }
    command -v openssl >/dev/null 2>&1 || { log_error "OpenSSL is not installed."; exit 1; }

    log_info "All prerequisites met."
}

# =============================================================================
# Create Kind cluster
# =============================================================================
create_cluster() {
    log_info "Creating Kind cluster: $KIND_CLUSTER_NAME"

    if kind get clusters | grep -q "^${KIND_CLUSTER_NAME}$"; then
        log_info "Using existing cluster: $KIND_CLUSTER_NAME"
        return 0
    fi

    kind create cluster --name "$KIND_CLUSTER_NAME" --config "$KIND_DIR/kind-config.yaml" --wait 5m

    log_info "Kind cluster created successfully!"
}

recreate_cluster() {
    delete_cluster
    create_cluster
}

# =============================================================================
# Delete Kind cluster
# =============================================================================
delete_cluster() {
    log_info "Deleting Kind cluster: $KIND_CLUSTER_NAME"
    kind delete cluster --name "$KIND_CLUSTER_NAME" 2>/dev/null || true
    log_info "Cluster deleted."
}

# =============================================================================
# Deploy infrastructure (MySQL, Redis, RabbitMQ) - NO Etcd
# =============================================================================
deploy_infrastructure() {
    log_info "Deploying infrastructure (K8s native - no Etcd)..."

    kubectl apply -f "$KIND_DIR/namespace.yaml"
    ensure_secrets

    log_info "Deploying MySQL..."
    kubectl apply -f "$KIND_DIR/mysql-statefulset.yaml"
    kubectl rollout status statefulset/mysql -n ecommerce --timeout=300s

    log_info "Deploying Redis Cluster..."
    kubectl delete job redis-cluster-init -n ecommerce --ignore-not-found=true
    kubectl apply -f "$KIND_DIR/redis-cluster.yaml"
    kubectl rollout status statefulset/redis -n ecommerce --timeout=300s
    kubectl wait --for=condition=complete job/redis-cluster-init -n ecommerce --timeout=180s

    log_info "Deploying RabbitMQ..."
    kubectl apply -f "$KIND_DIR/rabbitmq.yaml"
    kubectl rollout status deployment/rabbitmq -n ecommerce --timeout=300s

    log_info "Waiting for infrastructure to be ready..."
    sleep 10

    log_info "Infrastructure deployed!"
}

ensure_secrets() {
    log_info "Ensuring local development secrets..."

    kubectl create secret generic ecommerce-secrets \
        --namespace ecommerce \
        --from-literal=mysql-password=root123456 \
        --from-literal=redis-password=redis123456 \
        --from-literal=rabbitmq-user=guest \
        --from-literal=rabbitmq-password=guest \
        --dry-run=client -o yaml | kubectl apply -f -

    if ! kubectl get secret jwt-secret -n ecommerce >/dev/null 2>&1; then
        local jwt_dir
        jwt_dir=$(mktemp -d)
        openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$jwt_dir/private.pem" >/dev/null 2>&1
        openssl rsa -pubout -in "$jwt_dir/private.pem" -out "$jwt_dir/public.pem" >/dev/null 2>&1
        kubectl create secret generic jwt-secret \
            --namespace ecommerce \
            --from-file=private.pem="$jwt_dir/private.pem" \
            --from-file=public.pem="$jwt_dir/public.pem"
        rm -rf "$jwt_dir"
    fi
}

# =============================================================================
# Initialize database
# =============================================================================
init_database() {
    log_info "Initializing database..."

    # Check if init.sql exists
    local init_sql="$SCRIPT_DIR/../sql/init.sql"
    local migrations_dir="$SCRIPT_DIR/../sql/migrations"
    if [ ! -f "$init_sql" ]; then
        log_warn "Init SQL not found at $init_sql"
        return 0
    fi

    # Wait for MySQL to be ready
    log_info "Waiting for MySQL to be ready..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if kubectl exec -n ecommerce mysql-0 -- mysqladmin ping -uroot -proot123456 &>/dev/null; then
            log_info "MySQL is ready!"
            break
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    echo ""

    if [ $attempt -eq $max_attempts ]; then
        log_error "MySQL failed to start within timeout."
        return 1
    fi

    # Create database
    log_info "Creating database and tables..."
    kubectl exec -n ecommerce mysql-0 -- mysql -uroot -proot123456 -e "CREATE DATABASE IF NOT EXISTS ecommerce_demo CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null || true

    # 新库导入完整 schema；已有库则只执行未应用的增量迁移。
    local order_table_exists
    order_table_exists=$(kubectl exec -n ecommerce mysql-0 -- mysql -uroot -proot123456 ecommerce_demo -Nse "SHOW TABLES LIKE 'order';" 2>/dev/null)
    if [ -z "$order_table_exists" ]; then
        log_info "Importing schema..."
        kubectl exec -i -n ecommerce mysql-0 -- mysql -uroot -proot123456 ecommerce_demo < "$init_sql"
    else
        log_info "Existing schema detected."
    fi

    kubectl exec -n ecommerce mysql-0 -- mysql -uroot -proot123456 ecommerce_demo -e \
        "CREATE TABLE IF NOT EXISTS schema_migrations (name varchar(255) PRIMARY KEY, applied_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP);"

    for migration in "$migrations_dir"/*.sql; do
        [ -e "$migration" ] || break
        local migration_name
        migration_name=$(basename "$migration")

        if [ -z "$order_table_exists" ]; then
            kubectl exec -n ecommerce mysql-0 -- mysql -uroot -proot123456 ecommerce_demo -e \
                "INSERT IGNORE INTO schema_migrations(name) VALUES ('$migration_name');"
            continue
        fi

        local applied
        applied=$(kubectl exec -n ecommerce mysql-0 -- mysql -uroot -proot123456 ecommerce_demo -Nse \
            "SELECT COUNT(*) FROM schema_migrations WHERE name = '$migration_name';")
        if [ "$applied" = "0" ]; then
            log_info "Applying migration: $migration_name"
            kubectl exec -i -n ecommerce mysql-0 -- mysql -uroot -proot123456 ecommerce_demo < "$migration"
            kubectl exec -n ecommerce mysql-0 -- mysql -uroot -proot123456 ecommerce_demo -e \
                "INSERT INTO schema_migrations(name) VALUES ('$migration_name');"
        fi
    done

    log_info "Database initialized!"
}

# =============================================================================
# Deploy application services
# =============================================================================
deploy_services() {
    log_info "Deploying application services..."

    for svc in gateway user product cart order payment address stock; do
        log_info "Deploying $svc..."
        kubectl apply -f "$KIND_DIR/services/${svc}.yaml"
    done

    log_info "Deploying Prometheus and Grafana..."
    kubectl apply -f "$KIND_DIR/services/alerting-rules.yaml"
    kubectl apply -f "$KIND_DIR/services/prometheus.yaml"
    kubectl apply -f "$KIND_DIR/services/grafana.yaml"

    # Kind uses locally loaded `latest` images with imagePullPolicy=Never.
    # Applying an unchanged Deployment does not recreate Pods, so explicitly
    # restart them to pick up images most recently loaded into containerd.
    log_info "Restarting application deployments to pick up locally loaded images..."
    for svc in gateway user product cart order payment address stock order-delay order-cron order-dlq; do
        kubectl rollout restart "deployment/$svc" -n ecommerce
    done

    log_info "Waiting for deployments to be ready..."
    for svc in gateway user product cart order payment address stock order-delay order-cron order-dlq; do
        echo -n "Checking $svc..."
        kubectl rollout status "deployment/$svc" -n ecommerce --timeout=120s 2>/dev/null || log_warn "$svc rollout timeout"
        echo ""
    done

    log_info "Application services deployed!"
}

# =============================================================================
# Show deployment status
# =============================================================================
status() {
    log_info "E-commerce Deployment Status (K8s Native)"
    echo ""
    echo -e "${BLUE}===============================================${NC}"
    echo -e "${BLUE}  Infrastructure (no Etcd)${NC}"
    echo -e "${BLUE}===============================================${NC}"
    kubectl get pods -n ecommerce -l 'app in (mysql,redis,rabbitmq)'
    echo ""
    echo -e "${BLUE}===============================================${NC}"
    echo -e "${BLUE}  Application Services${NC}"
    echo -e "${BLUE}===============================================${NC}"
    kubectl get pods -n ecommerce -l 'app in (gateway,user,product,cart,order,payment,address,stock,order-delay,order-cron,order-dlq)'
    echo ""
    echo -e "${BLUE}===============================================${NC}"
    echo -e "${BLUE}  Services${NC}"
    echo -e "${BLUE}===============================================${NC}"
    kubectl get svc -n ecommerce
    echo ""
    echo -e "${GREEN}Gateway:   http://localhost:30088${NC}"
    echo -e "${GREEN}Prometheus:http://localhost:30909${NC}"
    echo -e "${GREEN}Grafana:   http://localhost:30300${NC}"
    echo -e "${GREEN}RabbitMQ:  http://localhost:31672${NC}"
}

# =============================================================================
# Full deployment
# =============================================================================
full_deploy() {
    check_prerequisites
    create_cluster
    deploy_infrastructure
    init_database
    deploy_services
    status
}

# =============================================================================
# Help
# =============================================================================
help() {
    cat << EOF

===============================================
  Kind Deployment Script (K8s Native)
===============================================

Note: No Etcd - using K8s native DNS for service discovery

Usage: ./deploy-kind.sh [command]

Commands:
  create       Create Kind cluster only
  recreate     Delete and recreate the Kind cluster
  delete       Delete Kind cluster
  infra        Deploy infrastructure only
  secrets      Create or refresh local development secrets
  init         Initialize or migrate the database
  services     Deploy application services only
  full         Full deployment (cluster + infra + services)
  status       Show deployment status
  logs [svc]   Show logs for a service
  clean        Clean up all resources

===============================================

EOF
}

case "${1:-help}" in
    create)
        check_prerequisites
        create_cluster
        ;;
    recreate)
        check_prerequisites
        recreate_cluster
        ;;
    delete)
        delete_cluster
        ;;
    infra)
        deploy_infrastructure
        ;;
    secrets)
        kubectl apply -f "$KIND_DIR/namespace.yaml"
        ensure_secrets
        ;;
    init)
        init_database
        ;;
    services)
        deploy_services
        ;;
    full)
        full_deploy
        ;;
    status)
        status
        ;;
    logs)
        kubectl logs -n ecommerce -l "app=${2:-gateway}" --tail=100 -f
        ;;
    clean)
        log_warn "Cleaning up all resources..."
        kubectl delete namespace ecommerce --ignore-not-found=true
        delete_cluster
        log_info "Cleanup complete!"
        ;;
    help|*)
        help
        ;;
esac
