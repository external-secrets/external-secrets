locals {
  tags = {
    Environment = "development"
    Owner       = "external-secrets"
    Repository  = "external-secrets"
    Purpose     = "managed e2e tests"
  }
}

module "cluster" {
  source = "./modules/cluster"

  cluster_name      = var.AWS_CLUSTER_NAME
  cluster_region    = var.AWS_REGION
  irsa_sa_name      = var.AWS_SA_NAME
  irsa_sa_namespace = var.AWS_SA_NAMESPACE
  tags              = local.tags
}
