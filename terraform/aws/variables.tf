variable "AWS_SA_NAME" {
  type    = string
  default = "eso-e2e-test"
}

variable "AWS_SA_NAMESPACE" {
  type    = string
  default = "default"
}

variable "AWS_REGION" {
  type    = string
  default = "eu-west-1"
}

variable "AWS_CLUSTER_NAME" {
  type    = string
  default = "eso-e2e-managed"
}
