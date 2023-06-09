/******************************************
  Default DNS Policy
 *****************************************/

resource "google_dns_policy" "default_policy" {
  project                   = var.project_id
  name                      = "dp-${var.environment_code}-shared-base-default-policy"
  enable_inbound_forwarding = var.dns_enable_inbound_forwarding
  enable_logging            = var.dns_enable_logging
  networks {
    network_url = module.main.network_self_link
  }
}

/******************************************
 Creates DNS Peering to DNS HUB
*****************************************/
data "google_compute_network" "vpc_dns_hub" {
  name    = "vpc-c-dns-hub"
  project = var.dns_hub_project_id
}

module "peering_zone" {
  source  = "terraform-google-modules/cloud-dns/google"
  version = "~> 5.0"

  project_id  = var.project_id
  type        = "peering"
  name        = "dz-${var.environment_code}-shared-base-to-dns-hub"
  domain      = var.domain
  description = "Private DNS peering zone."

  private_visibility_config_networks = [
    module.main.network_self_link
  ]
  target_network = data.google_compute_network.vpc_dns_hub.self_link
}
