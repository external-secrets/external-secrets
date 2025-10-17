resource "google_service_account" "default" {
  project    = var.project_id
  account_id = "e2e-managed-secretmanager"
}

resource "google_project_iam_member" "secretadmin" {
  project = var.project_id
  role    = "roles/secretmanager.admin"
  member  = "serviceAccount:${google_service_account.default.email}"
}

resource "google_project_iam_member" "service_account_token_creator" {
  project = var.project_id
  role    = "roles/iam.serviceAccountTokenCreator"
  member  = "serviceAccount:${google_service_account.default.email}"
}

resource "google_service_account_iam_member" "pod_identity" {
  for_each           = toset(var.workload_identity_users)
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[default/${each.value}]"
  service_account_id = google_service_account.default.name
}

resource "google_container_cluster" "primary" {
  project             = var.project_id
  name                = var.cluster_name
  initial_node_count  = 1
  network             = var.network
  subnetwork          = var.subnetwork
  location            = var.region
  deletion_protection = false

  ip_allocation_policy {}
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }
}

