module "private_service_connect" {
  source  = "terraform-google-modules/network/google//modules/private-service-connect"
  version = "~> 5.2"

  dns_code                   = "dz-${var.environment_code}-shared-base"
  forwarding_rule_target     = "all-apis"
  network_self_link          = module.main.network_self_link
  private_service_connect_ip = var.private_service_connect_ip
  project_id                 = var.project_id
}
