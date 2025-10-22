terraform {
  backend "gcs" {
    bucket = "eso-e2e-tfstate"
    prefix = "gcp-infrastructure"
  }

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 7.5"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 7.5"
    }
  }
}

provider "google" {
  project = "external-secrets-operator"
  region  = "europe-west1"
  zone    = "europe-west1-b"
}

provider "google-beta" {
  project = "external-secrets-operator"
  region  = "europe-west1"
  zone    = "europe-west1-b"
}
