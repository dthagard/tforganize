/**
 * Copyright 2022 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
