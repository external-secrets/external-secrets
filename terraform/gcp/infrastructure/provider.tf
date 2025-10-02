terraform {
  backend "gcs" {
    bucket      = "eso-infra-state"
    prefix      = "eso-infra-state/state"
    # TODO above bucket/prefix configuration is valid for the old account
    # the new account w/ identity federation should use the below bucket.
    #bucket      = "eso-e2e-tfstate"
    credentials = "../secrets/gcloud-service-account-key.json"
  }

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
