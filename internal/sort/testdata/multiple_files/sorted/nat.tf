resource "google_compute_address" "nat_external_addresses_region1" {
  count = var.nat_enabled ? var.nat_num_addresses_region1 : 0

  name    = "ca-${local.vpc_name}-${var.default_region1}-${count.index}"
  project = var.project_id
  region  = var.default_region1
}

resource "google_compute_address" "nat_external_addresses_region2" {
  count = var.nat_enabled ? var.nat_num_addresses_region2 : 0

  name    = "ca-${local.vpc_name}-${var.default_region2}-${count.index}"
  project = var.project_id
  region  = var.default_region2
}

/******************************************
  NAT Cloud Router & NAT config
 *****************************************/

resource "google_compute_router" "nat_router_region1" {
  count = var.nat_enabled ? 1 : 0

  bgp {
    asn = var.nat_bgp_asn
  }

  name    = "cr-${local.vpc_name}-${var.default_region1}-nat-router"
  network = module.main.network_self_link
  project = var.project_id
  region  = var.default_region1
}

resource "google_compute_router" "nat_router_region2" {
  count = var.nat_enabled ? 1 : 0

  bgp {
    asn = var.nat_bgp_asn
  }

  name    = "cr-${local.vpc_name}-${var.default_region2}-nat-router"
  network = module.main.network_self_link
  project = var.project_id
  region  = var.default_region2
}

resource "google_compute_router_nat" "egress_nat2" {
  count = var.nat_enabled ? 1 : 0

  log_config {
    enable = true
    filter = "TRANSLATIONS_ONLY"
  }

  name                               = "rn-${local.vpc_name}-${var.default_region2}-egress"
  nat_ip_allocate_option             = "MANUAL_ONLY"
  nat_ips                            = google_compute_address.nat_external_addresses_region2.*.self_link
  project                            = var.project_id
  region                             = var.default_region2
  router                             = google_compute_router.nat_router_region2[0].name
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

resource "google_compute_router_nat" "egress_nat_region1" {
  count = var.nat_enabled ? 1 : 0

  log_config {
    enable = true
    filter = "TRANSLATIONS_ONLY"
  }

  name                               = "rn-${local.vpc_name}-${var.default_region1}-egress"
  nat_ip_allocate_option             = "MANUAL_ONLY"
  nat_ips                            = google_compute_address.nat_external_addresses_region1.*.self_link
  project                            = var.project_id
  region                             = var.default_region1
  router                             = google_compute_router.nat_router_region1[0].name
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}
