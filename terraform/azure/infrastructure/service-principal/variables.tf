variable "application_display_name" {
  type        = string
  description = "Metadata name to use."
}
variable "application_owners" {
  type = list(string)
}
variable "issuer" {
  type = string
}
variable "audiences" {
  type    = list(string)
  default = ["api://AzureADTokenExchange"]
}
variable "subject" {
  type = string
}
