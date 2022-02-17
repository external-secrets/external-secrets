resource "google_compute_network" "env-vpc" {
  project                 = var.project_id
  name                    = "${var.env}-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "env-subnet" {
  project       = var.project_id
  name          = "${google_compute_network.env-vpc.name}-subnet"
  region        = var.region
  network       = google_compute_network.env-vpc.name
  ip_cidr_range = "10.10.0.0/24"
}

output "vpc-name" {
  value = google_compute_network.env-vpc.name
}
output "vpc-id" {
  value = google_compute_network.env-vpc.id
}
output "vpc-object" {
  value = google_compute_network.env-vpc.self_link
}
output "subnet-name" {
  value = google_compute_subnetwork.env-subnet.name
}
output "subnet-ip_cidr_range" {
  value = google_compute_subnetwork.env-subnet.ip_cidr_range
}
