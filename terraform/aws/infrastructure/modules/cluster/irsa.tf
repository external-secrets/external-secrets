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

# Create the IAM policy document for SSM Parameter Store access
data "aws_iam_policy_document" "ssm_parameterstore" {
  statement {
    actions = [
      "ssm:GetParameter*",
      "ssm:PutParameter",
      "ssm:DescribeParameters",
      "ssm:DeleteParameter*",
      "ssm:AddTagsToResource",
      "ssm:ListTagsForResource",
      "ssm:RemoveTagsFromResource",
      "tag:GetResources"
    ]
    effect    = "Allow"
    resources = ["*"]
  }
}

resource "aws_iam_role" "eso-e2e-irsa" {
  name               = "eso-e2e-irsa"
  path               = "/"
  assume_role_policy = data.aws_iam_policy_document.assume-policy.json
}

# Attach the AWS managed policy for Secrets Manager
resource "aws_iam_role_policy_attachment" "secrets_manager" {
  role       = aws_iam_role.eso-e2e-irsa.name
  policy_arn = "arn:aws:iam::aws:policy/SecretsManagerReadWrite"
}

# Create and attach the inline policy for SSM Parameter Store
resource "aws_iam_role_policy" "ssm_parameterstore" {
  name   = "aws_ssm_parameterstore"
  role   = aws_iam_role.eso-e2e-irsa.id
  policy = data.aws_iam_policy_document.ssm_parameterstore.json
}
