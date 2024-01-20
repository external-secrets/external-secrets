resource "azuread_application" "current" {
  display_name = var.application_display_name
  owners       = var.application_owners
}

resource "azuread_service_principal" "current" {
  application_id               = azuread_application.current.application_id
  app_role_assignment_required = false
  owners                       = var.application_owners
  feature_tags {
    enterprise = true
    gallery    = true
  }
}

resource "azuread_service_principal_password" "current" {
  service_principal_id = azuread_service_principal.current.id
}

resource "azuread_application_federated_identity_credential" "example" {
  application_object_id = azuread_application.current.object_id
  display_name          = var.application_display_name
  audiences             = var.audiences
  issuer                = var.issuer
  subject               = var.subject
}
