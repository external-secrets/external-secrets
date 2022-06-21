variable "cluster_name" {
  type        = string
  description = "The name of the Managed Kubernetes Cluster to create"
}

variable "resource_group_name" {
  type        = string
  description = "The Name which should be used for this Resource Group"
}

variable "resource_group_location" {
  type        = string
  description = "The Azure Region where the Resource Group should exist"
}

variable "dns_prefix" {
  type        = string
  description = "DNS prefix specified when creating the managed cluster"
  default     = "api"
}

variable "oidc_issuer_enabled" {
  type        = bool
  description = "Enable or Disable the OIDC issuer URL"
  default     = true
}

variable "default_node_pool_name" {
  type        = string
  description = " The name of the Default Node Pool which should be created within the Kubernetes Cluster"
  default     = "default"
}

variable "default_node_pool_node_count" {
  type        = number
  description = " The initial number of nodes which should exist within this Node Pool"
}

variable "default_node_pool_vm_size" {
  type        = string
  description = " The SKU which should be used for the Virtual Machines used in this Node Pool"

}

variable "cluster_tags" {
  type        = map(string)
  description = "A mapping of tags to assign to the cluster"
}
