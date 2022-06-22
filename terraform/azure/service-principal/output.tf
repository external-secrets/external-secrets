output "application_id" {
  value = azuread_application.current.application_id
}
output "sp_id" {
  value = azuread_service_principal.current.id
}
output "sp_object_id" {
  value = azuread_service_principal.current.object_id
}
