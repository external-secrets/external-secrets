terraform {
  required_providers {
    azapi = {
      source  = "azure/azapi"
      version = ">= 2.0, < 3.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 5.0, < 6.0"
    }
  }
}
