data "azurerm_client_config" "current" {}

data "azurerm_subscription" "primary" {}

module "test_resource_group" {
  source = "./resource-group"

  resource_group_name     = var.resource_group_name
  resource_group_location = var.resource_group_location
}

module "test_sp" {
  source = "./service-principal"

  application_display_name = var.application_display_name
  application_owners       = [data.azurerm_client_config.current.object_id]
  issuer                   = module.test_aks.cluster_issuer_url
  subject                  = "system:serviceaccount:${var.sa_namespace}:${var.sa_name}"
}

module "test_key_vault" {
  source = "./key-vault"

  key_vault_display_name  = var.key_vault_display_name
  resource_group_location = var.resource_group_location
  resource_group_name     = var.resource_group_name
  tenant_id               = data.azurerm_client_config.current.tenant_id
  client_object_id        = data.azurerm_client_config.current.object_id
  eso_sp_object_id        = module.test_sp.sp_object_id
}

module "test_workload_identity" {
  source = "./workload-identity"

  tenant_id = data.azurerm_client_config.current.tenant_id
  tags      = var.cluster_tags

}

module "test_aks" {
  source = "./aks"

  cluster_name                 = var.cluster_name
  resource_group_name          = var.resource_group_name
  resource_group_location      = var.resource_group_location
  default_node_pool_node_count = var.default_node_pool_node_count
  default_node_pool_vm_size    = var.default_node_pool_vm_size
  cluster_tags                 = var.cluster_tags
}

resource "azurerm_role_assignment" "current" {
  scope                = data.azurerm_subscription.primary.id
  role_definition_name = "Reader"
  principal_id         = module.test_sp.sp_id
}

resource "azurerm_key_vault_secret" "test" {
  name         = "secret-sauce"
  value        = "szechuan"
  key_vault_id = module.test_key_vault.key_vault_id
}
