terraform {
  backend "azurerm" {
    resource_group_name  = "external-secrets-tfstate-rg"
    storage_account_name = "esoe2emanagedtfstate"
    container_name       = "tfstate"
    key                  = "infrastructure/terraform.tfstate"
  }
  required_providers {
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
  }
}

provider "azurerm" {
  features {}
  # set this to false when running locally
  use_oidc = false
}
