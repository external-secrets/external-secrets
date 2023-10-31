variable "key_vault_display_name" {
  type        = string
  description = "Metadata name to use."
}
variable "resource_group_name" {
  type        = string
  description = "The Name which should be used for this Resource Group"
}
variable "resource_group_location" {
  type        = string
  description = "The Azure Region where the Resource Group should exist"
}
variable "tenant_id" {
  type        = string
  description = "Azure Tenant ID"
}
variable "client_object_id" {
  type        = string
  description = "The object ID of a user, service principal or security group in the Azure Active Directory tenant for the vault"
}
variable "eso_sp_object_id" {
  type        = string
  description = "The object ID of the ESO service account"
}

variable "eso_e2e_sp_object_id" {
  type        = string
  description = "The object ID of the ESO e2e service account"
}
