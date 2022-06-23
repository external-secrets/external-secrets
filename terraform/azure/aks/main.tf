
resource "azurerm_kubernetes_cluster" "current" {
  name                = var.cluster_name
  resource_group_name = var.resource_group_name
  location            = var.resource_group_location
  dns_prefix          = var.dns_prefix

  oidc_issuer_enabled               = var.oidc_issuer_enabled
  role_based_access_control_enabled = true

  default_node_pool {
    name       = var.default_node_pool_name
    node_count = var.default_node_pool_node_count
    vm_size    = var.default_node_pool_vm_size
  }

  identity {
    type = "SystemAssigned"
  }

  tags = var.cluster_tags

}
