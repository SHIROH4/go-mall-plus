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

KIND_CLUSTER_NAME="ecommerce-cluster"

# =============================================================================
# Check prerequisites
# =============================================================================
check_prerequisites() {
    log_info "Checking prerequisites..."

    command -v kind >/dev/null 2>&1 || { log_error "Kind is not installed."; exit 1; }
    command -v kubectl >/dev/null 2>&1 || { log_error "kubectl is not installed."; exit 1; }
    command -v docker >/dev/null 2>&1 || { log_error "Docker is not installed."; exit 1; }

    log_info "All prerequisites met."
}

# =============================================================================
# Create Kind cluster
# =============================================================================
create_cluster() {
    log_info "Creating Kind cluster: $KIND_CLUSTER_NAME"

    if kind get clusters | grep -q "^${KIND_CLUSTER_NAME}$"; then
        log_warn "Cluster $KIND_CLUSTER_NAME already exists."
        read -p "Delete and recreate? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            delete_cluster
        else
            log_info "Using existing cluster."
            return 0
        fi
    fi

    kind create cluster --config "$KIND_DIR/kind-config.yaml" --wait 5m

    log_info "Kind cluster created successfully!"
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
    kubectl apply -f "$KIND_DIR/secrets.yaml"

    log_info "Deploying MySQL..."
    kubectl apply -f "$KIND_DIR/mysql.yaml"
    kubectl rollout status deployment/mysql -n ecommerce --timeout=300s

    log_info "Deploying Redis..."
    kubectl apply -f "$KIND_DIR/redis.yaml"
    kubectl rollout status deployment/redis -n ecommerce --timeout=180s

    log_info "Deploying RabbitMQ..."
    kubectl apply -f "$KIND_DIR/rabbitmq.yaml"
    kubectl rollout status deployment/rabbitmq -n ecommerce --timeout=180s

    log_info "Waiting for infrastructure to be ready..."
    sleep 10

    log_info "Infrastructure deployed!"
}

# =============================================================================
# Initialize database
# =============================================================================
init_database() {
    log_info "Initializing database..."

    # Check if init.sql exists
    local init_sql="$SCRIPT_DIR/../../../sql/init.sql"
    if [ ! -f "$init_sql" ]; then
        log_warn "Init SQL not found at $init_sql"
        return 0
    fi

    # Wait for MySQL to be ready
    log_info "Waiting for MySQL to be ready..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if kubectl exec -n ecommerce deployment/mysql -- mysqladmin ping -uroot -proot123456 &>/dev/null; then
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
    kubectl exec -n ecommerce deployment/mysql -- mysql -uroot -proot123456 -e "CREATE DATABASE IF NOT EXISTS ecommerce_demo CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null || true

    # Check if tables already exist
    local tables_exist=$(kubectl exec -n ecommerce deployment/mysql -- mysql -uroot -proot123456 ecommerce_demo -e "SHOW TABLES;" 2>/dev/null | wc -l)
    if [ "$tables_exist" -gt 0 ]; then
        log_info "Tables already exist, skipping init."
    else
        log_info "Importing schema..."
        kubectl exec -n ecommerce deployment/mysql -- mysql -uroot -proot123456 ecommerce_demo < "$init_sql" 2>/dev/null || \
            log_warn "Failed to import SQL. Please manually initialize the database."
    fi

    log_info "Database initialized!"
}

# =============================================================================
# Deploy application services
# =============================================================================
deploy_services() {
    log_info "Deploying application services..."

    for svc in gateway user product cart order payment address; do
        log_info "Deploying $svc..."
        kubectl apply -f "$KIND_DIR/services/${svc}.yaml"
    done

    log_info "Waiting for deployments to be ready..."
    for svc in gateway user product cart order payment address order-delay order-cron order-dlq; do
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
    kubectl get pods -n ecommerce -l 'app in (gateway,user,product,cart,order,payment,address,order-delay,order-cron,order-dlq)'
    echo ""
    echo -e "${BLUE}===============================================${NC}"
    echo -e "${BLUE}  Services${NC}"
    echo -e "${BLUE}===============================================${NC}"
    kubectl get svc -n ecommerce
    echo ""
    echo -e "${GREEN}Gateway:   http://localhost:30088${NC}"
    echo -e "${GREEN}Frontend:  http://localhost:30080${NC}"
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
  delete       Delete Kind cluster
  infra        Deploy infrastructure only
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
    delete)
        delete_cluster
        ;;
    infra)
        deploy_infrastructure
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
