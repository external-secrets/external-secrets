#!/usr/bin/env bash

set -euo pipefail

# Colors for output
BLUE='\033[34m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="${NAMESPACE:-external-secrets}"
HELM_RELEASE="${HELM_RELEASE:-external-secrets}"
HELM_CHART="./deploy/charts/external-secrets"

# Function to check and install missing dependencies
check_and_install_deps() {
    local missing_deps=()
    local can_auto_install=false

    # Detect OS
    OS=$(uname -s)

    # Check if we have a package manager available
    if [ "$OS" = "Darwin" ] && command -v brew &> /dev/null; then
        can_auto_install=true
        PKG_MGR="brew"
    fi

    # Check for required commands
    command -v kubectl &> /dev/null || missing_deps+=("kubectl")
    command -v helm &> /dev/null || missing_deps+=("helm")
    command -v docker &> /dev/null || missing_deps+=("docker")
    command -v make &> /dev/null || missing_deps+=("make")
    command -v go &> /dev/null || missing_deps+=("go")
    command -v yq &> /dev/null || missing_deps+=("yq")

    if [ ${#missing_deps[@]} -eq 0 ]; then
        return 0
    fi

    echo -e "${YELLOW}[WARN]${NC} Missing dependencies: ${missing_deps[*]}"

    if [ "$can_auto_install" = true ]; then
        echo -e "${BLUE}[INFO]${NC} Homebrew detected. Installing missing dependencies..."
        for dep in "${missing_deps[@]}"; do
            case $dep in
                kubectl)
                    echo -e "${BLUE}[INFO]${NC} Installing kubectl..."
                    brew install kubectl
                    ;;
                helm)
                    echo -e "${BLUE}[INFO]${NC} Installing helm..."
                    brew install helm
                    ;;
                docker)
                    echo -e "${RED}[ERROR]${NC} Docker not found. Please install Docker Desktop from https://www.docker.com/products/docker-desktop"
                    exit 1
                    ;;
                make)
                    echo -e "${RED}[ERROR]${NC} Make not found. Please install Xcode Command Line Tools: xcode-select --install"
                    exit 1
                    ;;
                go)
                    echo -e "${BLUE}[INFO]${NC} Installing go..."
                    brew install go
                    ;;
                yq)
                    echo -e "${BLUE}[INFO]${NC} Installing yq..."
                    brew install yq
                    ;;
            esac
        done
        echo -e "${GREEN}[SUCCESS]${NC} Dependencies installed successfully"
    else
        echo -e "${RED}[ERROR]${NC} Missing required dependencies: ${missing_deps[*]}"
        echo -e "${RED}[ERROR]${NC} Please install them manually:"
        for dep in "${missing_deps[@]}"; do
            case $dep in
                kubectl)
                    echo -e "  - kubectl: https://kubernetes.io/docs/tasks/tools/"
                    ;;
                helm)
                    echo -e "  - helm: https://helm.sh/docs/intro/install/"
                    ;;
                docker)
                    echo -e "  - docker: https://docs.docker.com/get-docker/"
                    ;;
                make)
                    echo -e "  - make: usually available via build-essential or similar package"
                    ;;
                go)
                    echo -e "  - go: https://go.dev/doc/install"
                    ;;
                yq)
                    echo -e "  - yq: https://github.com/mikefarah/yq#install"
                    ;;
            esac
        done
        exit 1
    fi
}

# Check dependencies first
echo -e "${BLUE}[INFO]${NC} Checking dependencies..."
check_and_install_deps

# Detect the target architecture for the k8s cluster
# For kind/k3d/minikube, we need to build for the container node architecture
if command -v kind &> /dev/null && kind get clusters 2>/dev/null | grep -q .; then
    # For kind, check the node architecture
    NODE_ARCH=$(kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}')
    ARCH="${NODE_ARCH}"
elif command -v k3d &> /dev/null && k3d cluster list 2>/dev/null | grep -v "No clusters found" &> /dev/null; then
    # For k3d, check the node architecture
    NODE_ARCH=$(kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}')
    ARCH="${NODE_ARCH}"
elif command -v minikube &> /dev/null && minikube status &> /dev/null; then
    # For minikube, check the node architecture
    NODE_ARCH=$(kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}')
    ARCH="${NODE_ARCH}"
else
    # For other setups (Docker Desktop, Rancher, Colima), use host architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
    esac
fi

echo -e "${BLUE}[INFO]${NC} Building external-secrets from current branch..."
echo -e "${BLUE}[INFO]${NC} Target architecture: ${ARCH}"

# Step 1: Generate code and CRDs
echo -e "${BLUE}[INFO]${NC} Generating code and CRDs..."
make generate

# Step 2: Build the binary for the current architecture
echo -e "${BLUE}[INFO]${NC} Building binary for ${ARCH}..."
make build-${ARCH}

# Step 3: Get the version tag
VERSION=$(make docker.tag)
IMAGE_NAME=$(make docker.imagename)
IMAGE_TAG="${IMAGE_NAME}:${VERSION}"

echo -e "${BLUE}[INFO]${NC} Image will be tagged as: ${IMAGE_TAG}"

# Step 4: Build the Docker image
echo -e "${BLUE}[INFO]${NC} Building Docker image..."
make docker.build ARCH="${ARCH}"

# Step 5: Load the image into the local k8s cluster
echo -e "${BLUE}[INFO]${NC} Loading image into local k8s cluster..."

# Detect which k8s cluster type is being used
if command -v kind &> /dev/null && kind get clusters 2>/dev/null | grep -q .; then
    CLUSTER=$(kubectl config current-context | sed 's/kind-//')
    echo -e "${YELLOW}[INFO]${NC} Detected kind cluster: ${CLUSTER}"
    kind load docker-image "${IMAGE_TAG}" --name "${CLUSTER}"
elif command -v minikube &> /dev/null && minikube status &> /dev/null; then
    echo -e "${YELLOW}[INFO]${NC} Detected minikube cluster"
    minikube image load "${IMAGE_TAG}"
elif command -v k3d &> /dev/null && k3d cluster list 2>/dev/null | grep -v "No clusters found" &> /dev/null; then
    CLUSTER=$(kubectl config current-context | sed 's/k3d-//')
    echo -e "${YELLOW}[INFO]${NC} Detected k3d cluster: ${CLUSTER}"
    k3d image import "${IMAGE_TAG}" -c "${CLUSTER}"
elif docker context inspect rancher-desktop &> /dev/null && kubectl config current-context | grep -q rancher; then
    echo -e "${YELLOW}[INFO]${NC} Detected Rancher Desktop - image already available"
elif docker context inspect colima &> /dev/null || (docker context show | grep -q colima); then
    echo -e "${YELLOW}[INFO]${NC} Detected Colima - image already available"
else
    echo -e "${YELLOW}[WARN]${NC} Could not detect k8s cluster type (kind/minikube/k3d/rancher-desktop/colima)"
    echo -e "${YELLOW}[WARN]${NC} Assuming Docker Desktop or similar where images are automatically available"
fi

# Step 6: Install CRDs first to ensure they're ready before controller starts
echo -e "${BLUE}[INFO]${NC} Installing CRDs with server-side apply..."
kubectl create namespace "${NAMESPACE}" 2>/dev/null || true
make crds.install

# Wait for CRDs to be established
echo -e "${BLUE}[INFO]${NC} Waiting for CRDs to be established..."
kubectl wait --for condition=established --timeout=60s crd/externalsecrets.external-secrets.io 2>/dev/null || true
kubectl wait --for condition=established --timeout=60s crd/secretstores.external-secrets.io 2>/dev/null || true
kubectl wait --for condition=established --timeout=60s crd/clustersecretstores.external-secrets.io 2>/dev/null || true

# Step 7: Generate helm manifests to ensure they're up to date
echo -e "${BLUE}[INFO]${NC} Generating helm chart dependencies..."
helm dependency update "${HELM_CHART}"

# Step 8: Deploy or upgrade using Helm
echo -e "${BLUE}[INFO]${NC} Deploying to namespace '${NAMESPACE}' as release '${HELM_RELEASE}'..."

# Check if the release already exists
if helm list -n "${NAMESPACE}" | grep -q "${HELM_RELEASE}"; then
    echo -e "${BLUE}[INFO]${NC} Upgrading existing release..."
    helm upgrade "${HELM_RELEASE}" "${HELM_CHART}" \
        --namespace "${NAMESPACE}" \
        --set image.repository="${IMAGE_NAME%:*}" \
        --set image.tag="${VERSION}" \
        --set image.pullPolicy=Never \
        --set installCRDs=false \
        --wait
else
    echo -e "${BLUE}[INFO]${NC} Installing new release..."
    kubectl create namespace "${NAMESPACE}" 2>/dev/null || true
    helm install "${HELM_RELEASE}" "${HELM_CHART}" \
        --namespace "${NAMESPACE}" \
        --set image.repository="${IMAGE_NAME%:*}" \
        --set image.tag="${VERSION}" \
        --set image.pullPolicy=Never \
        --set installCRDs=false \
        --wait
fi

# Step 9: Verify the deployment
echo -e "${BLUE}[INFO]${NC} Verifying deployment..."
kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/name=external-secrets

echo -e "${GREEN}[SUCCESS]${NC} External Secrets deployed successfully!"
echo -e "${GREEN}[SUCCESS]${NC} Image: ${IMAGE_TAG}"
echo -e "${GREEN}[SUCCESS]${NC} Release: ${HELM_RELEASE} in namespace ${NAMESPACE}"
echo ""
echo -e "${BLUE}[INFO]${NC} To view logs, run:"
echo -e "  kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/name=external-secrets -f"
