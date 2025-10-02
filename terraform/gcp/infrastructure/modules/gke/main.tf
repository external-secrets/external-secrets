resource "google_service_account" "default" {
  project    = var.project_id
  account_id = var.GCP_GSA_NAME
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
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[default/${var.GCP_KSA_NAME}]"
  service_account_id = google_service_account.default.name
}

resource "google_service_account_iam_member" "pod_identity_e2e" {
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[default/external-secrets-e2e]"
  service_account_id = google_service_account.default.name
}

resource "google_container_cluster" "primary" {
  project            = var.project_id
  name               = "e2e"
  initial_node_count = 1
  network            = var.network
  subnetwork         = var.subnetwork
  location           = var.region

  ip_allocation_policy {}
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }
}

