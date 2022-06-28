
resource "azurerm_key_vault" "current" {
  name                        = var.key_vault_display_name
  location                    = var.resource_group_location
  resource_group_name         = var.resource_group_name
  enabled_for_disk_encryption = true
  tenant_id                   = var.tenant_id
  soft_delete_retention_days  = 7
  purge_protection_enabled    = false

  sku_name = "standard"

  access_policy {
    tenant_id = var.tenant_id
    object_id = var.client_object_id

    key_permissions = [
      "Get",
    ]

    secret_permissions = [
      "Set",
      "Get",
      "Delete",
      "Purge",
      "Recover"
    ]

    storage_permissions = [
      "Get",
    ]
  }
  access_policy {
    tenant_id = var.tenant_id
    object_id = var.eso_sp_object_id

    secret_permissions = [
      "Get",
    ]

  }
}
