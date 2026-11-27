output "cluster_issuer_url" {
  value = azurerm_kubernetes_cluster.current.oidc_issuer_url
}

output "cluster_id" {
  value = azurerm_kubernetes_cluster.current.id
}
