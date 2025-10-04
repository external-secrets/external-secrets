data "azurerm_client_config" "current" {}

data "azurerm_subscription" "primary" {}

locals {
  resource_group_name = "external-secrets-e2e"
}

resource "azurerm_resource_group" "current" {
  name     = local.resource_group_name
  location = var.resource_group_location
}

module "test_sp" {
  source = "./service-principal"

  application_display_name = "managed-e2e-suite-external-secrets-operator"
  application_owners       = [data.azurerm_client_config.current.object_id]
  issuer                   = module.test_aks.cluster_issuer_url
  subject                  = "system:serviceaccount:${var.sa_namespace}:${var.sa_name}"

  depends_on = [
    azurerm_resource_group.current
  ]
}

module "e2e_sp" {
  source = "./service-principal"

  application_display_name = "managed-e2e-suite-external-secrets-e2e"
  application_owners       = [data.azurerm_client_config.current.object_id]
  issuer                   = module.test_aks.cluster_issuer_url
  subject                  = "system:serviceaccount:default:external-secrets-e2e"
}

module "test_key_vault" {
  source = "./key-vault"

  key_vault_display_name  = var.key_vault_display_name
  resource_group_location = var.resource_group_location
  resource_group_name     = local.resource_group_name
  tenant_id               = data.azurerm_client_config.current.tenant_id
  client_object_id        = data.azurerm_client_config.current.object_id
  eso_sp_object_id        = module.test_sp.sp_object_id
  eso_e2e_sp_object_id    = module.e2e_sp.sp_object_id

  depends_on = [
    azurerm_resource_group.current
  ]
}


module "test_aks" {
  source = "./aks"

  cluster_name                 = var.cluster_name
  resource_group_name          = local.resource_group_name
  resource_group_location      = var.resource_group_location
  default_node_pool_node_count = var.default_node_pool_node_count
  default_node_pool_vm_size    = var.default_node_pool_vm_size
  cluster_tags                 = var.cluster_tags

  depends_on = [
    azurerm_resource_group.current
  ]
}

resource "azurerm_role_assignment" "current" {
  scope                = data.azurerm_subscription.primary.id
  role_definition_name = "Owner"
  principal_id         = module.test_sp.sp_id

  depends_on = [
    azurerm_resource_group.current
  ]
}
