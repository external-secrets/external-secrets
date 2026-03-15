resource "kubernetes_namespace" "eso" {
  metadata {
    name = "external-secrets-operator"
  }
}

data "azurerm_client_config" "current" {}

data "azuread_application" "eso" {
  display_name = "managed-e2e-suite-external-secrets-operator"
}

data "azuread_application" "e2e" {
  display_name = "managed-e2e-suite-external-secrets-e2e"
}

// the `e2e` pod itself runs with workload identity and
// does not rely on client credentials.
resource "kubernetes_service_account" "e2e" {
  metadata {
    name      = "external-secrets-e2e"
    namespace = "default"
    annotations = {
      "azure.workload.identity/client-id" = data.azuread_application.e2e.client_id
    }
    labels = {
      "azure.workload.identity/use" = "true"
    }
  }
  depends_on = [kubernetes_namespace.eso]
}

resource "kubernetes_service_account" "current" {
  metadata {
    name      = "external-secrets-operator"
    namespace = "external-secrets-operator"
    annotations = {
      "azure.workload.identity/client-id" = data.azuread_application.eso.client_id
    }
    labels = {
      "azure.workload.identity/use" = "true"
    }
  }
  depends_on = [kubernetes_namespace.eso]
}
