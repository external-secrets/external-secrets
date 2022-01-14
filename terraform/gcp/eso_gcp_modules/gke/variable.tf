variable "project_id" {
  default = "my-project-1475718618821"
}
variable "env" {
  default = "dev"
}
variable "region" {
  default = "europe-west1"
}
variable "zone" {
  default = "europe-west1-b"
}
variable "zones" {
  default = ["europe-west1-a", "europe-west1-b", "europe-west1-c"]
}
variable "network" {
  default = "dev-vpc"
}
variable "subnetwork" {
  default = "dev-subnetwork"
}
variable "ip_pod_range" {
  default = "dev-pod-ip-range"
}
variable "ip_service_range" {
  default = "dev-service-ip-range"
}
variable "horizontal_pod_autoscaling" {
  default = false
}
variable "node_count" {
  default = 2
}
variable "node_min_count" {
  default = 2
}
variable "node_max_count" {
  default = 2
}
variable "initial_node_count" {
  default = 2
}
variable "preemptible" {
  default = true
}

variable "GCP_GSA_NAME" {type = string}
variable "GCP_KSA_NAME" {type = string}
