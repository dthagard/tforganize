/******************************************
 Creates DNS Peering to DNS HUB
*****************************************/
data "google_compute_network" "vpc_dns_hub" {
  name    = "vpc-c-dns-hub"
  project = var.dns_hub_project_id
}

/******************************************
  Default DNS Policy
 *****************************************/

resource "google_dns_policy" "default_policy" {
  enable_inbound_forwarding = var.dns_enable_inbound_forwarding
  enable_logging            = var.dns_enable_logging
  name                      = "dp-${var.environment_code}-shared-base-default-policy"

  networks {
    network_url = module.main.network_self_link
  }

  project = var.project_id
}

module "peering_zone" {
  source  = "terraform-google-modules/cloud-dns/google"
  version = "~> 5.0"

  description = "Private DNS peering zone."
  domain      = var.domain
  name        = "dz-${var.environment_code}-shared-base-to-dns-hub"
  private_visibility_config_networks = [
    module.main.network_self_link
  ]
  project_id     = var.project_id
  target_network = data.google_compute_network.vpc_dns_hub.self_link
  type           = "peering"
}
