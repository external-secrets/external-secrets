// must match IAM Role in infrastructure/modules/cluster 
data "aws_iam_role" "eso-e2e-irsa" {
  name = "eso-e2e-irsa"
}

resource "kubernetes_service_account" "this" {
  metadata {
    name      = var.AWS_SA_NAME
    namespace = var.AWS_SA_NAMESPACE
    annotations = {
      "eks.amazonaws.com/role-arn" = data.aws_iam_role.eso-e2e-irsa.arn
    }
  }
}
