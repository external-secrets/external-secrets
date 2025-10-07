terraform {
  required_version = ">= 0.13"

  backend "azurerm" {
    resource_group_name  = "external-secrets-tfstate-rg"
    storage_account_name = "esoe2emanagedtfstate"
    container_name       = "tfstate"
    key                  = "kubernetes/terraform.tfstate"
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
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0"
    }
  }
}


provider "azurerm" {
  features {}
  subscription_id = "9cb8d43c-2ed5-40e7-aec8-76a177c32c15"
}


data "azurerm_kubernetes_cluster" "this" {
  name                = var.cluster_name
  resource_group_name = "external-secrets-e2e"
}

provider "helm" {
  kubernetes = {
    host                   = data.azurerm_kubernetes_cluster.this.kube_config[0].host
    username               = data.azurerm_kubernetes_cluster.this.kube_config[0].username
    password               = data.azurerm_kubernetes_cluster.this.kube_config[0].password
    client_certificate     = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].client_certificate)
    client_key             = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].client_key)
    cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].cluster_ca_certificate)
  }
}
provider "kubernetes" {
  host                   = data.azurerm_kubernetes_cluster.this.kube_config[0].host
  username               = data.azurerm_kubernetes_cluster.this.kube_config[0].username
  password               = data.azurerm_kubernetes_cluster.this.kube_config[0].password
  client_certificate     = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].client_certificate)
  client_key             = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].client_key)
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].cluster_ca_certificate)
}
