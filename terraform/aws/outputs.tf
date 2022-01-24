output "cluster_arn" {
  value = module.cluster.cluster_arn
}

output "cluster_iam_role_arn" {
  value = module.cluster.cluster_iam_role_arn
}

output "aws_auth_configmap_yaml" {
  value = module.cluster.aws_auth_configmap_yaml
}
