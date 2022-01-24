variable "env" {
  default = "dev"
}
variable "ip_cidr_range" {
  default = "10.69.0.0/16"
}
variable "ip_pod_range" {
  default = "10.70.0.0/16"
}
variable "ip_service_range" {
  default = "10.71.0.0/16"
}
variable "region" {
  default = "europe-west1"
}
variable "project_id" {
  type = string
}
