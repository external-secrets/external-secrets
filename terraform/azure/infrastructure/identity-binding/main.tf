// AKS identity binding maps a user-assigned managed identity (UAMI) to the
// whole cluster using a single federated identity credential, instead of one
// credential per <namespace, service-account> pair. This module provisions the
// UAMI and the identity binding so the controller pod can authenticate using
// `workloadIdentityMethod: IdentityBinding`.

resource "azurerm_user_assigned_identity" "current" {
  name                = var.identity_name
  resource_group_name = var.resource_group_name
  location            = var.resource_group_location
}

resource "azapi_resource" "identity_binding" {
  type      = var.az_ib_type
  name      = var.identity_binding_name
  parent_id = var.cluster_id

  body = {
    properties = {
      managedIdentity = {
        resourceId = azurerm_user_assigned_identity.current.id
      }
    }
  }
}
