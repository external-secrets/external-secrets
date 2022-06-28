variable "cluster_name" {
  type        = string
  description = "The name of the Managed Kubernetes Cluster to create"
  default     = "eso-cluster"
}

variable "resource_group_name" {
  type        = string
  description = "The Name which should be used for this Resource Group"
  default     = "external-secrets-operator"
}

variable "resource_group_location" {
  type        = string
  description = "The Azure Region where the Resource Group should exist"
  default     = "westeurope"
}
variable "application_display_name" {
  type        = string
  description = "Metadata name to use."
  default     = "external-secrets-operator"
}

variable "dns_prefix" {
  type        = string
  description = "DNS prefix specified when creating the managed cluster"
  default     = "eso"
}

variable "key_vault_display_name" {
  type        = string
  description = "The name of the Key Vault to create"
  default     = "eso-testing"
}

variable "default_node_pool_name" {
  type        = string
  description = " The name of the Default Node Pool which should be created within the Kubernetes Cluster"
  default     = "default"
}

variable "default_node_pool_node_count" {
  type        = number
  description = " The initial number of nodes which should exist within this Node Pool"
  default     = 1
}

variable "default_node_pool_vm_size" {
  type        = string
  description = " The SKU which should be used for the Virtual Machines used in this Node Pool"
  default     = "Standard_B2ms"
}
variable "sa_name" {
  type    = string
  default = "external-secrets-operator"
}
variable "sa_namespace" {
  type        = string
  description = "The namespace where the service account will be created"
  default     = "external-secrets-operator"
}
variable "cluster_tags" {
  type        = map(string)
  description = "A mapping of tags to assign to the cluster"
  default     = { cluster_name = "external-secrets-operator" }
}
