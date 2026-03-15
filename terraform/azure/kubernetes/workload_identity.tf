resource "kubernetes_namespace" "azure-workload-identity-system" {
  metadata {
    annotations = {
      name = "azure-workload-identity-system"
    }
    name = "azure-workload-identity-system"
  }
}

resource "helm_release" "azure-workload-identity-system" {
  name       = "workload-identity-webhook"
  namespace  = "azure-workload-identity-system"
  chart      = "workload-identity-webhook"
  repository = "https://azure.github.io/azure-workload-identity/charts"
  wait       = true
  depends_on = [kubernetes_namespace.azure-workload-identity-system]

  set = [{
    name  = "azureTenantID"
    value = data.azurerm_client_config.current.tenant_id
  }]
}
