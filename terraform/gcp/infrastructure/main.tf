locals {
  credentials_path = "secrets/gcloud-service-account-key.json"
  region           = "europe-west1"
}

module "network" {
  source     = "./modules/network"
  region     = local.region
  project_id = var.GCP_PROJECT_ID
}

module "cluster" {
  source       = "./modules/gke"
  project_id   = var.GCP_PROJECT_ID
  region       = local.region
  network      = module.network.network_name
  subnetwork   = module.network.subnetwork_name
  GCP_GSA_NAME = var.GCP_GSA_NAME
  GCP_KSA_NAME = var.GCP_KSA_NAME
}
