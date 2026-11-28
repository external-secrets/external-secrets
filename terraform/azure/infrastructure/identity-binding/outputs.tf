output "client_id" {
  value = azurerm_user_assigned_identity.current.client_id
}

output "principal_id" {
  value = azurerm_user_assigned_identity.current.principal_id
}

output "tenant_id" {
  value = azurerm_user_assigned_identity.current.tenant_id
}
