resource "google_compute_firewall" "allow_all_egress" {
  count = var.allow_all_egress_ranges != null ? 1 : 0

  allow {
    protocol = "all"
  }

  destination_ranges = var.allow_all_egress_ranges
  direction          = "EGRESS"

  dynamic "log_config" {
    for_each = var.firewall_enable_logging == true ? [{
      metadata = "INCLUDE_ALL_METADATA"
    }] : []

    content {
      metadata = log_config.value.metadata
    }
  }

  name     = "fw-${var.environment_code}-shared-base-1000-e-a-all-all-all"
  network  = module.main.network_name
  priority = 1000
  project  = var.project_id
}

resource "google_compute_firewall" "allow_all_ingress" {
  count = var.allow_all_ingress_ranges != null ? 1 : 0

  allow {
    protocol = "all"
  }

  direction = "INGRESS"

  dynamic "log_config" {
    for_each = var.firewall_enable_logging == true ? [{
      metadata = "INCLUDE_ALL_METADATA"
    }] : []

    content {
      metadata = log_config.value.metadata
    }
  }

  name          = "fw-${var.environment_code}-shared-base-1000-i-a-all"
  network       = module.main.network_name
  priority      = 1000
  project       = var.project_id
  source_ranges = var.allow_all_ingress_ranges
}

resource "google_compute_firewall" "allow_private_api_egress" {
  allow {
    ports    = ["443"]
    protocol = "tcp"
  }

  destination_ranges = [local.private_googleapis_cidr]
  direction          = "EGRESS"

  dynamic "log_config" {
    for_each = var.firewall_enable_logging == true ? [{
      metadata = "INCLUDE_ALL_METADATA"
    }] : []

    content {
      metadata = log_config.value.metadata
    }
  }

  name        = "fw-${var.environment_code}-shared-base-65530-e-a-allow-google-apis-all-tcp-443"
  network     = module.main.network_name
  priority    = 65530
  project     = var.project_id
  target_tags = ["allow-google-apis"]
}

/******************************************
  Mandatory firewall rules
 *****************************************/

resource "google_compute_firewall" "deny_all_egress" {
  deny {
    protocol = "all"
  }

  destination_ranges = ["0.0.0.0/0"]
  direction          = "EGRESS"

  dynamic "log_config" {
    for_each = var.firewall_enable_logging == true ? [{
      metadata = "INCLUDE_ALL_METADATA"
    }] : []

    content {
      metadata = log_config.value.metadata
    }
  }

  name     = "fw-${var.environment_code}-shared-base-65530-e-d-all-all-all"
  network  = module.main.network_name
  priority = 65530
  project  = var.project_id
}
