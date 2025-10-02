resource "kubernetes_service_account" "test" {
  metadata {
    name = var.GCP_KSA_NAME
    annotations = {
      "iam.gke.io/gcp-service-account" : "${var.GCP_GSA_NAME}@${var.GCP_PROJECT_ID}.iam.gserviceaccount.com"
    }
  }
}
