terraform {
  required_version = ">= 0.13"

  backend "s3" {
    bucket = "eso-tfstate-e2e-managed"
    key    = "aws-tfstate"
    region = "eu-central-1"
  }

  required_providers {}
}

provider "aws" {
  region = "eu-central-1"

  default_tags {
    tags = {
      Environment = "development"
      Owner       = "external-secrets"
      Repository  = "external-secrets"
      Purpose     = "managed e2e tests"
    }
  }
}
