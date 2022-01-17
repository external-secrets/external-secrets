provider "google" {
  project = "external-secrets-operator"
  region = "europe-west1"
  zone = "europe-west1-b"
  credentials = file(var.credentials_path)
}

provider "google-beta" {
  project = "external-secrets-operator"
  region = "europe-west1"
  zone = "europe-west1-b"
  credentials = file(var.credentials_path)
}
