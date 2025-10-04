variable "project_id" {
  type = string
}
variable "region" {
  type = string
}
variable "network" {
  type = string
}
variable "subnetwork" {
  type = string
}
variable "workload_identity_users" {
  type = list(string)
}
variable "cluster_name" {
  type = string
}