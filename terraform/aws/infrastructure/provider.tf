terraform {
  required_version = ">= 0.13"

  backend "s3" {
    bucket = "eso-tfstate-e2e-managed"
    key    = "aws-tfstate"
    region = "eu-central-1"
  }
}
