terraform {
  required_version = ">= 0.13"

  backend "s3" {
    bucket = "eso-tfstate-e2e-managed"
    key    = "aws-tfstate-kubernetes"
    region = "eu-central-1"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
  }
}

provider "aws" {
  region = var.AWS_REGION
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.this.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.this.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.this.token
}

data "aws_eks_cluster_auth" "this" {
  name = var.AWS_CLUSTER_NAME
}
data "aws_eks_cluster" "this" {
  name = var.AWS_CLUSTER_NAME
}
