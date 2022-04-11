locals {
  sa_manifest = <<-EOT
      apiVersion: v1
      kind: ServiceAccount
      metadata:
        name: ${local.serviceaccount_name}
        namespace: ${local.serviceaccount_namespace}
        annotations:
          eks.amazonaws.com/role-arn: "${aws_iam_role.eso-e2e-irsa.arn}"
  EOT
}

data "aws_iam_policy_document" "assume-policy" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    condition {
      test     = "StringEquals"
      variable = "${trimprefix(module.eks.cluster_oidc_issuer_url, "https://")}:sub"

      values = [
        "system:serviceaccount:${local.serviceaccount_namespace}:${local.serviceaccount_name}"
      ]
    }

    principals {
      type        = "Federated"
      identifiers = [module.eks.oidc_provider_arn]
    }
  }
}

resource "aws_iam_role" "eso-e2e-irsa" {
  name               = "eso-e2e-irsa"
  path               = "/"
  assume_role_policy = data.aws_iam_policy_document.assume-policy.json
  managed_policy_arns = [
    "arn:aws:iam::aws:policy/SecretsManagerReadWrite"
  ]

  inline_policy {
    name = "aws_ssm_parameterstore"

    policy = jsonencode({
      Version = "2012-10-17"
      Statement = [
        {
          Action = [
            "ssm:GetParameter",
            "ssm:PutParameter",
            "ssm:DescribeParameters",
          ]
          Effect   = "Allow"
          Resource = "*"
        },
      ]
    })
  }


}

resource "null_resource" "apply_sa" {
  triggers = {
    kubeconfig = base64encode(local.kubeconfig)
    cmd_patch  = <<-EOT
      echo '${local.sa_manifest}' | kubectl --kubeconfig <(echo $KUBECONFIG | base64 --decode) apply -f -
    EOT
  }

  provisioner "local-exec" {
    interpreter = ["/bin/bash", "-c"]
    environment = {
      KUBECONFIG = self.triggers.kubeconfig
    }
    command = self.triggers.cmd_patch
  }
}
