#!/bin/bash
#
# Uninstall External Secrets Operator V2 E2E installation
# This script removes the monolithic Helm chart installation
#

set -e

NAMESPACE="external-secrets-system"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_info "Uninstalling External Secrets Operator V2"

# Delete all ExternalSecret resources first (they have finalizers that need to be removed by the controller)
log_info "Deleting all ExternalSecret resources (waiting for finalizers to be processed)"
kubectl delete externalsecrets --all --all-namespaces --timeout=120s 2>/dev/null || log_warning "No ExternalSecrets found or already deleted"

# Delete other resources that may have finalizers
log_info "Deleting all PushSecret resources"
kubectl delete pushsecrets --all --all-namespaces --timeout=120s 2>/dev/null || log_warning "No PushSecrets found or already deleted"

log_info "Deleting all ClusterExternalSecret resources"
kubectl delete clusterexternalsecrets --all --timeout=120s 2>/dev/null || log_warning "No ClusterExternalSecrets found or already deleted"

log_info "Deleting all ClusterPushSecret resources"
kubectl delete clusterpushsecrets --all --timeout=120s 2>/dev/null || log_warning "No ClusterPushSecrets found or already deleted"

# Uninstall the monolithic Helm release
log_info "Removing Helm release: external-secrets"
helm uninstall external-secrets -n "$NAMESPACE" 2>/dev/null || log_warning "Helm release 'external-secrets' not found"

# Delete any leftover resources
log_info "Cleaning up any leftover resources"

# Delete CRDs (only if you want to clean them completely)
log_info "Deleting CRDs"
kubectl delete crd secretstores.external-secrets.io 2>/dev/null || true
kubectl delete crd clustersecretstores.external-secrets.io 2>/dev/null || true
kubectl delete crd externalsecrets.external-secrets.io 2>/dev/null || true
kubectl delete crd clusterexternalsecrets.external-secrets.io 2>/dev/null || true
kubectl delete crd pushsecrets.external-secrets.io 2>/dev/null || true
kubectl delete crd clusterpushsecrets.external-secrets.io 2>/dev/null || true
kubectl delete crd generators.external-secrets.io 2>/dev/null || true
kubectl delete crd clustergenerators.external-secrets.io 2>/dev/null || true

# Delete namespace
log_info "Deleting namespace: $NAMESPACE"
kubectl delete namespace "$NAMESPACE" --ignore-not-found=true --timeout=60s

log_info "Uninstallation complete"
