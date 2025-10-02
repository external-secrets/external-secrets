terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 3.5"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 3.5"
    }
  }
}

provider "google" {
  project     = "external-secrets-operator"
  region      = "europe-west1"
  zone        = "europe-west1-b"
  credentials = file("../secrets/gcloud-service-account-key.json")
}

provider "google-beta" {
  project     = "external-secrets-operator"
  region      = "europe-west1"
  zone        = "europe-west1-b"
  credentials = file("../secrets/gcloud-service-account-key.json")
}


data "google_client_config" "default" {}

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.this.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(data.google_container_cluster.this.master_auth.0.cluster_ca_certificate)
}


data "google_container_cluster" "this" {
  project  = var.GCP_PROJECT_ID
  location = "europe-west1-b" # must match ../infrastructure
  name     = "e2e"
}
