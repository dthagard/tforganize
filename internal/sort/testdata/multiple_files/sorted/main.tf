/***************************************************************
  VPC Peering Configuration
 **************************************************************/

data "google_compute_network" "vpc_base_net_hub" {
  count = var.mode == "spoke" ? 1 : 0

  name    = "vpc-c-shared-base-hub"
  project = var.base_net_hub_project_id
}

locals {
  mode                    = var.mode == "hub" ? "-hub" : "-spoke"
  network_name            = "vpc-${local.vpc_name}"
  private_googleapis_cidr = module.private_service_connect.private_service_connect_ip
  vpc_name                = "${var.environment_code}-shared-base${local.mode}"
}

/******************************************
  Shared VPC configuration
 *****************************************/

module "main" {
  source  = "terraform-google-modules/network/google"
  version = "~> 5.1"

  delete_default_internet_gateway_routes = "true"
  network_name                           = local.network_name
  project_id                             = var.project_id
  routes = concat(
    var.nat_enabled ?
    [
      {
        name              = "rt-${local.vpc_name}-1000-egress-internet-default"
        description       = "Tag based route through IGW to access internet"
        destination_range = "0.0.0.0/0"
        tags              = "egress-internet"
        next_hop_internet = "true"
        priority          = "1000"
      }
    ]
    : [],
    var.windows_activation_enabled ?
    [{
      name              = "rt-${local.vpc_name}-1000-all-default-windows-kms"
      description       = "Route through IGW to allow Windows KMS activation for GCP."
      destination_range = "35.190.247.13/32"
      next_hop_internet = "true"
      priority          = "1000"
      }
    ]
    : []
  )
  secondary_ranges = var.secondary_ranges
  shared_vpc_host  = "true"
  subnets          = var.subnets
}

module "peering" {
  source  = "terraform-google-modules/network/google//modules/network-peering"
  version = "~> 5.1"
  count   = var.mode == "spoke" ? 1 : 0

  export_peer_custom_routes = true
  local_network             = module.main.network_self_link
  peer_network              = data.google_compute_network.vpc_base_net_hub[0].self_link
  prefix                    = "np"
}

/************************************
  Router to advertise shared VPC
  subnetworks and Google Private API
************************************/

module "region1_router1" {
  source  = "terraform-google-modules/cloud-router/google"
  version = "~> 4.0"
  count   = var.mode != "spoke" ? 1 : 0

  bgp = {
    asn                  = var.bgp_asn_subnet
    advertised_groups    = ["ALL_SUBNETS"]
    advertised_ip_ranges = [{ range = local.private_googleapis_cidr }]
  }
  name    = "cr-${local.vpc_name}-${var.default_region1}-cr1"
  network = module.main.network_name
  project = var.project_id
  region  = var.default_region1
}

module "region1_router2" {
  source  = "terraform-google-modules/cloud-router/google"
  version = "~> 4.0"
  count   = var.mode != "spoke" ? 1 : 0

  bgp = {
    asn                  = var.bgp_asn_subnet
    advertised_groups    = ["ALL_SUBNETS"]
    advertised_ip_ranges = [{ range = local.private_googleapis_cidr }]
  }
  name    = "cr-${local.vpc_name}-${var.default_region1}-cr2"
  network = module.main.network_name
  project = var.project_id
  region  = var.default_region1
}

module "region2_router1" {
  source  = "terraform-google-modules/cloud-router/google"
  version = "~> 4.0"
  count   = var.mode != "spoke" ? 1 : 0

  bgp = {
    asn                  = var.bgp_asn_subnet
    advertised_groups    = ["ALL_SUBNETS"]
    advertised_ip_ranges = [{ range = local.private_googleapis_cidr }]
  }
  name    = "cr-${local.vpc_name}-${var.default_region2}-cr3"
  network = module.main.network_name
  project = var.project_id
  region  = var.default_region2
}

module "region2_router2" {
  source  = "terraform-google-modules/cloud-router/google"
  version = "~> 4.0"
  count   = var.mode != "spoke" ? 1 : 0

  bgp = {
    asn                  = var.bgp_asn_subnet
    advertised_groups    = ["ALL_SUBNETS"]
    advertised_ip_ranges = [{ range = local.private_googleapis_cidr }]
  }
  name    = "cr-${local.vpc_name}-${var.default_region2}-cr4"
  network = module.main.network_name
  project = var.project_id
  region  = var.default_region2
}

/***************************************************************
  Configure Service Networking for Cloud SQL & future services.
 **************************************************************/

resource "google_compute_global_address" "private_service_access_address" {
  count = var.private_service_cidr != null ? 1 : 0

  address       = element(split("/", var.private_service_cidr), 0)
  address_type  = "INTERNAL"
  name          = "ga-${local.vpc_name}-vpc-peering-internal"
  network       = module.main.network_self_link
  prefix_length = element(split("/", var.private_service_cidr), 1)
  project       = var.project_id
  purpose       = "VPC_PEERING"

  depends_on = [module.peering]
}

resource "google_service_networking_connection" "private_vpc_connection" {
  count = var.private_service_cidr != null ? 1 : 0

  network                 = module.main.network_self_link
  reserved_peering_ranges = [google_compute_global_address.private_service_access_address[0].name]
  service                 = "servicenetworking.googleapis.com"

  depends_on = [module.peering]
}
