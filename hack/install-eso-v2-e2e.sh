#!/bin/bash
#
# Install External Secrets Operator V2 for E2E testing
# This script deploys the controller and Kubernetes provider using the monolithic Helm chart
#
# Prerequisites:
#   - kubectl and helm installed
#   - Access to a Kubernetes cluster (kind recommended for local testing)
#   - Docker images built and available:
#     * ghcr.io/external-secrets/external-secrets:latest
#     * ghcr.io/external-secrets/provider-kubernetes:latest
#
# For kind clusters, images will be automatically loaded if available locally.
#
# Build images before running (if not already built):
#   make docker.build VERSION=latest
#   # This builds:
#   #   - Controller: ghcr.io/external-secrets/external-secrets:latest
#   #   - Kubernetes Provider: ghcr.io/external-secrets/provider-kubernetes:latest
#   #   - AWS Provider: ghcr.io/external-secrets/provider-aws:latest
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CHARTS_DIR="$ROOT_DIR/deploy/charts"
NAMESPACE="external-secrets-system"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites"
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found"
        exit 1
    fi
    
    if ! command -v helm &> /dev/null; then
        log_error "helm not found"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Detect if running in kind cluster
is_kind_cluster() {
    kubectl config current-context | grep -q "kind-"
}

# Get kind cluster name from context
get_kind_cluster_name() {
    kubectl config current-context | sed 's/kind-//'
}

# Load Docker images into kind cluster
load_images_to_kind() {
    if ! is_kind_cluster; then
        log_info "Not a kind cluster, skipping image loading"
        return 0
    fi
    
    if ! command -v kind &> /dev/null; then
        log_warning "kind CLI not found, cannot load images"
        log_warning "Please ensure images are available in the cluster"
        return 0
    fi
    
    local cluster_name
    cluster_name=$(get_kind_cluster_name)
    
    log_info "Detected kind cluster: $cluster_name"
    log_info "Loading Docker images into kind cluster"
    
    # Controller image
    local controller_image="ghcr.io/external-secrets/external-secrets:latest"
    if docker image inspect "$controller_image" &> /dev/null; then
        log_info "Loading controller image: $controller_image"
        kind load docker-image "$controller_image" --name "$cluster_name"
    else
        log_warning "Controller image not found locally: $controller_image"
        log_warning "Attempting to pull from registry (may fail if not published)"
    fi
    
    # Provider images
    local kubernetes_provider_image="ghcr.io/external-secrets/provider-kubernetes:latest"
    if docker image inspect "$kubernetes_provider_image" &> /dev/null; then
        log_info "Loading provider image: $kubernetes_provider_image"
        kind load docker-image "$kubernetes_provider_image" --name "$cluster_name"
    else
        log_warning "Provider image not found locally: $kubernetes_provider_image"
        log_warning "Attempting to pull from registry (may fail if not published)"
    fi
    
    local fake_provider_image="ghcr.io/external-secrets/provider-fake:latest"
    if docker image inspect "$fake_provider_image" &> /dev/null; then
        log_info "Loading provider image: $fake_provider_image"
        kind load docker-image "$fake_provider_image" --name "$cluster_name"
    else
        log_warning "Fake provider image not found locally: $fake_provider_image"
        log_warning "Attempting to pull from registry (may fail if not published)"
    fi

    local aws_provider_image="ghcr.io/external-secrets/provider-aws:latest"
    if docker image inspect "$aws_provider_image" &> /dev/null; then
        log_info "Loading provider image: $aws_provider_image"
        kind load docker-image "$aws_provider_image" --name "$cluster_name"
    else
        log_warning "aws provider image not found locally: $aws_provider_image"
        log_warning "Attempting to pull from registry (may fail if not published)"
    fi
    
    
    log_info "Image loading complete"
}

# Install External Secrets with Kubernetes provider using monolithic chart
install_external_secrets() {
    log_info "Installing External Secrets V2 with Kubernetes provider"
    
    # Create a temporary values file for the installation
    local values_file
    values_file=$(mktemp)
    
    cat > "$values_file" <<EOF
# Controller configuration
installCRDs: true
replicaCount: 1

image:
  repository: ghcr.io/external-secrets/external-secrets
  tag: latest
  pullPolicy: IfNotPresent

certController:
  image:
    repository: ghcr.io/external-secrets/external-secrets
    tag: latest
    pullPolicy: IfNotPresent

webhook:
  create: true
  image:
    repository: ghcr.io/external-secrets/external-secrets
    tag: latest
    pullPolicy: IfNotPresent

# Provider defaults configuration
providerDefaults:
  replicaCount: 1
  serviceAccount:
    create: true
    automount: true
  podSecurityContext:
    enabled: true
    runAsNonRoot: true
    runAsUser: 65532
    fsGroup: 65532
    seccompProfile:
      type: RuntimeDefault
  securityContext:
    enabled: true
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    runAsUser: 65532
    capabilities:
      drop:
      - ALL
  service:
    type: ClusterIP
    port: 8080
  resources:
    limits:
      cpu: 200m
      memory: 256Mi
    requests:
      cpu: 50m
      memory: 64Mi
  tls:
    enabled: true

# Enable provider deployments
providers:
  enabled: true
  list:
    - name: kubernetes
      type: kubernetes
      enabled: true
      image:
        repository: ghcr.io/external-secrets/provider-kubernetes
        tag: latest
        pullPolicy: IfNotPresent
    
    - name: fake
      type: fake
      enabled: true
      image:
        repository: ghcr.io/external-secrets/provider-fake
        tag: latest
        pullPolicy: IfNotPresent

    - name: aws
      type: aws
      enabled: true
      image:
        repository: ghcr.io/external-secrets/provider-aws
        tag: latest
        pullPolicy: IfNotPresent
      extraEnv:
      - name: AWS_SECRET_ACCESS_KEY
        value: "${AWS_SECRET_ACCESS_KEY}"
      - name: AWS_ACCESS_KEY_ID
        value: "${AWS_ACCESS_KEY_ID}"
      - name: AWS_SESSION_TOKEN
        value: "${AWS_SESSION_TOKEN}"
      - name: AWS_REGION
        value: "eu-central-1"

# Controller resources
resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 50m
    memory: 64Mi
EOF
    

    log_info "Installing with monolithic Helm chart"
    helm upgrade --install external-secrets "$CHARTS_DIR/external-secrets" \
        --create-namespace \
        --namespace "$NAMESPACE" \
        --values "$values_file" \
        --wait \
        --timeout 5m
    
    # Cleanup temporary file
    rm -f "$values_file"
    
    log_info "External Secrets with Kubernetes provider installed"

    kubectl -n "$NAMESPACE" delete po -l app.kubernetes.io/instance=external-secrets
}

# Verify installation
verify_installation() {
    log_info "Verifying installation"
    
    # Check controller pod
    log_info "Waiting for controller pod to be ready"
    if ! kubectl wait --for=condition=ready pod \
        -l app.kubernetes.io/name=external-secrets \
        -n "$NAMESPACE" \
        --timeout=300s; then
        log_error "Controller pod not ready"
        kubectl get pods -n "$NAMESPACE"
        kubectl describe pods -n "$NAMESPACE" -l app.kubernetes.io/name=external-secrets
        kubectl logs -n "$NAMESPACE" -l app.kubernetes.io/name=external-secrets --tail=50
        exit 1
    fi
    
    # Check Kubernetes provider pod
    log_info "Waiting for Kubernetes provider pod to be ready"
    if ! kubectl wait --for=condition=ready pod \
        -l "app.kubernetes.io/name=external-secrets-provider-kubernetes" \
        -n "$NAMESPACE" \
        --timeout=300s; then
        log_error "Kubernetes provider pod not ready"
        kubectl get pods -n "$NAMESPACE"
        kubectl describe pods -n "$NAMESPACE" -l app.kubernetes.io/name=external-secrets-provider-kubernetes
        kubectl logs -n "$NAMESPACE" -l app.kubernetes.io/name=external-secrets-provider-kubernetes --tail=50
        exit 1
    fi
    
    # Check Fake provider pod
    log_info "Waiting for Fake provider pod to be ready"
    if ! kubectl wait --for=condition=ready pod \
        -l "app.kubernetes.io/name=external-secrets-provider-fake" \
        -n "$NAMESPACE" \
        --timeout=300s; then
        log_error "Fake provider pod not ready"
        kubectl get pods -n "$NAMESPACE"
        kubectl describe pods -n "$NAMESPACE" -l app.kubernetes.io/name=external-secrets-provider-fake
        kubectl logs -n "$NAMESPACE" -l app.kubernetes.io/name=external-secrets-provider-fake --tail=50
        exit 1
    fi
    
    # Check cert controller pod
    log_info "Waiting for cert controller pod to be ready"
    if ! kubectl wait --for=condition=ready pod \
        -l app.kubernetes.io/name=external-secrets-cert-controller \
        -n "$NAMESPACE" \
        --timeout=300s; then
        log_warning "Cert controller pod not ready (may not be critical for testing)"
    fi
    
    log_info "All pods are ready"
    kubectl get pods -n "$NAMESPACE"
    
    # Show services
    log_info "Services:"
    kubectl get svc -n "$NAMESPACE"
}

# Main installation flow
main() {
    log_info "Installing External Secrets Operator V2 for E2E testing"
    log_info "Using monolithic Helm chart with Kubernetes provider"
    
    check_prerequisites
    load_images_to_kind
    install_external_secrets
    verify_installation
    
    log_info "Installation complete!"
    log_info ""
    log_info "Deployment summary:"
    log_info "  - Controller: external-secrets"
    log_info "  - Provider: kubernetes (integrated)"
    log_info "  - Namespace: $NAMESPACE"
    log_info ""
    log_info "Next steps:"
    log_info "  1. Run E2E tests: make test.e2e.v2"
    log_info "  2. View controller logs: kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=external-secrets -f"
    log_info "  3. View provider logs: kubectl logs -n $NAMESPACE -l app.kubernetes.io/component=provider -f"
    log_info "  4. Cleanup: ./hack/uninstall-eso-v2-e2e.sh"
}

main "$@"
