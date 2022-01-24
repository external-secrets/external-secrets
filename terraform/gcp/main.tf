terraform {
  backend "gcs" {
    bucket      = "eso-infra-state"
    prefix      = "eso-infra-state/state"
    credentials = "secrets/gcloud-service-account-key.json"
  }
}

module "test-network" {
  source        = "./eso_gcp_modules/network"
  env           = var.env
  region        = var.region
  ip_cidr_range = var.ip_cidr_range
  project_id    = var.GCP_PROJECT_ID
}

module "test-cluster" {
  source             = "./eso_gcp_modules/gke"
  project_id         = var.GCP_PROJECT_ID
  env                = var.env
  region             = var.region
  network            = module.test-network.vpc-object
  subnetwork         = module.test-network.subnet-name
  node_count         = var.node_count
  initial_node_count = var.initial_node_count
  preemptible        = true
  GCP_GSA_NAME       = var.GCP_GSA_NAME
  GCP_KSA_NAME       = var.GCP_KSA_NAME
}
