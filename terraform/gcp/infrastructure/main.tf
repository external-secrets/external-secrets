locals {
  credentials_path = "secrets/gcloud-service-account-key.json"
}

module "network" {
  source     = "./modules/network"
  region     = var.GCP_FED_REGION
  project_id = var.GCP_FED_PROJECT_ID
}

module "cluster" {
  source       = "./modules/gke"
  project_id   = var.GCP_FED_PROJECT_ID
  region       = var.GCP_FED_REGION
  cluster_name = var.GCP_GKE_CLUSTER
  network      = module.network.network_name
  subnetwork   = module.network.subnetwork_name

  workload_identity_users = [
    # eso provider which is set up by e2e tests to 
    # assert eso functionality.
    var.GCP_KSA_NAME,
    # e2e test runner which orchestrates the tests
    "external-secrets-e2e",
  ]
}
