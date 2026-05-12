#!/bin/bash
# =============================================================================
# Frontend Build and Deploy Script
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_ROOT="$SCRIPT_DIR/../.."
KIND_DIR="$SCRIPT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# =============================================================================
# Build frontend
# =============================================================================
build_frontend() {
    log_info "Building frontend..."

    cd "$FRONTEND_ROOT"

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log_info "Installing dependencies..."
        npm install
    fi

    # Build production assets
    log_info "Running npm build..."
    npm run build

    log_info "Frontend built successfully!"
}

# =============================================================================
# Build Docker image
# =============================================================================
build_docker() {
    log_info "Building Docker image..."

    local docker_dir="$KIND_DIR/docker"
    local context_dir="$KIND_DIR/context"

    # Create build context
    mkdir -p "$context_dir/dist"
    cp -r "$FRONTEND_ROOT/dist"/* "$context_dir/dist/" 2>/dev/null || true
    cp "$docker_dir/Dockerfile" "$context_dir/"
    cp "$docker_dir/nginx.conf" "$context_dir/"

    # Build image
    docker build -t ecommerce-frontend:latest -f "$docker_dir/Dockerfile" "$context_dir"

    # Load into Kind
    log_info "Loading image into Kind..."
    kind load docker-image ecommerce-frontend:latest --name ecommerce-cluster 2>/dev/null || \
        log_warn "Kind cluster not found. Run deploy-kind.sh first."

    # Cleanup context
    rm -rf "$context_dir"

    log_info "Docker image built and loaded!"
}

# =============================================================================
# Deploy to Kind
# =============================================================================
deploy_frontend() {
    log_info "Deploying frontend to Kind..."

    kubectl apply -f "$KIND_DIR/frontend.yaml"
    kubectl rollout status deployment/frontend -n ecommerce --timeout=120s

    log_info "Frontend deployed!"
}

# =============================================================================
# Full pipeline
# =============================================================================
full_deploy() {
    build_frontend
    build_docker
    deploy_frontend

    log_info ""
    log_info "==============================================="
    log_info "  Frontend Deployed!"
    log_info "==============================================="
    log_info ""
    log_info "Frontend:  http://localhost:30080"
    log_info ""
}

# =============================================================================
# Help
# =============================================================================
help() {
    cat << EOF

===============================================
  Frontend Deployment Script
===============================================

Usage: ./deploy-frontend.sh [command]

Commands:
  build         Build frontend only
  docker       Build Docker image only
  deploy       Deploy to Kind (requires image)
  full         Full pipeline: build + docker + deploy
  clean        Clean build artifacts

===============================================

EOF
}

case "${1:-help}" in
    build)
        build_frontend
        ;;
    docker)
        build_docker
        ;;
    deploy)
        deploy_frontend
        ;;
    full)
        full_deploy
        ;;
    clean)
        log_info "Cleaning build artifacts..."
        rm -rf "$FRONTEND_ROOT/dist"
        rm -rf "$KIND_DIR/context"
        log_info "Cleaned!"
        ;;
    help|*)
        help
        ;;
esac
