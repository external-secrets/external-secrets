resource "kubernetes_service_account" "test" {
  metadata {
    name = var.GCP_KSA_NAME
    annotations = {
      "iam.gke.io/gcp-service-account" : "e2e-managed-secretmanager@${var.GCP_FED_PROJECT_ID}.iam.gserviceaccount.com"
    }
  }
}
