variable "cluster_name" {
  type    = string
  default = "eso-e2e-managed"
}

variable "irsa_sa_name" {
  type = string
}

variable "irsa_sa_namespace" {
  type = string
}

variable "cluster_region" {
  type = string
}
