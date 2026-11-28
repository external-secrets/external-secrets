variable "identity_name" {
  type        = string
  description = "The name of the user-assigned managed identity to bind to the cluster"
}

variable "identity_binding_name" {
  type        = string
  description = "The name of the AKS identity binding"
}

variable "cluster_id" {
  type        = string
  description = "The ID of the AKS managed cluster"
}

variable "resource_group_name" {
  type        = string
  description = "The Name which should be used for this Resource Group"
}

variable "resource_group_location" {
  type        = string
  description = "The Azure Region where the Resource Group should exist"
}

variable "az_ib_type" {
  type    = string
  default = "Microsoft.ContainerService/managedClusters/identityBindings@2026-08-02-preview"
}
