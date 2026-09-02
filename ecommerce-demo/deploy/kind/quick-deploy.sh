#!/bin/bash
# =============================================================================
# Quick Deploy Script - Build and Deploy with single command
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

log_info "Starting quick deployment..."

# Step 1: Create or reuse cluster before images are loaded into Kind
log_info "Step 1/6: Creating or reusing Kind cluster..."
"$SCRIPT_DIR/deploy-kind.sh" create

# Step 2: Build
log_info "Step 2/6: Building binaries and Docker images..."
"$SCRIPT_DIR/build.sh" full

# Step 3: Load images into Kind before restarting deployments.
log_info "Step 3/6: Loading images into Kind..."
"$SCRIPT_DIR/build.sh" load

# Step 4: Deploy infrastructure
log_info "Step 4/6: Deploying infrastructure..."
"$SCRIPT_DIR/deploy-kind.sh" infra

# Step 5: Initialize or migrate database
log_info "Step 5/6: Initializing database..."
"$SCRIPT_DIR/deploy-kind.sh" init

# Step 6: Deploy services
log_info "Step 6/6: Deploying application services..."
"$SCRIPT_DIR/deploy-kind.sh" services

# Show status
log_info ""
log_info "==============================================="
log_info "  Deployment Complete!"
log_info "==============================================="
log_info ""
log_info "Gateway:   http://localhost:30088"
log_info "RabbitMQ:  http://localhost:31672"
log_info ""
log_info "Check status: ./deploy-kind.sh status"
log_info ""
