resource "google_compute_network" "vpc" {
  project                 = var.project_id
  name                    = "e2e"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  project       = var.project_id
  name          = "${google_compute_network.vpc.name}-subnet"
  region        = var.region
  network       = google_compute_network.vpc.name
  ip_cidr_range = "10.10.0.0/24"
}

