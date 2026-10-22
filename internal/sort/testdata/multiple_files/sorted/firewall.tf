resource "google_compute_firewall" "allow_all_egress" {
  count = var.allow_all_egress_ranges != null ? 1 : 0

  destination_ranges = var.allow_all_egress_ranges
  direction          = "EGRESS"
  name               = "fw-${var.environment_code}-shared-base-1000-e-a-all-all-all"
  network            = module.main.network_name
  priority           = 1000
  project            = var.project_id

  allow {
    protocol = "all"
  }

  dynamic "log_config" {
    for_each = var.firewall_enable_logging == true ? [{
      metadata = "INCLUDE_ALL_METADATA"
    }] : []

    content {
      metadata = log_config.value.metadata
    }
  }
}

resource "google_compute_firewall" "allow_all_ingress" {
  count = var.allow_all_ingress_ranges != null ? 1 : 0

  direction     = "INGRESS"
  name          = "fw-${var.environment_code}-shared-base-1000-i-a-all"
  network       = module.main.network_name
  priority      = 1000
  project       = var.project_id
  source_ranges = var.allow_all_ingress_ranges

  allow {
    protocol = "all"
  }

  dynamic "log_config" {
    for_each = var.firewall_enable_logging == true ? [{
      metadata = "INCLUDE_ALL_METADATA"
    }] : []

    content {
      metadata = log_config.value.metadata
    }
  }
}

resource "google_compute_firewall" "allow_private_api_egress" {
  destination_ranges = [local.private_googleapis_cidr]
  direction          = "EGRESS"
  name               = "fw-${var.environment_code}-shared-base-65530-e-a-allow-google-apis-all-tcp-443"
  network            = module.main.network_name
  priority           = 65530
  project            = var.project_id
  target_tags        = ["allow-google-apis"]

  allow {
    ports    = ["443"]
    protocol = "tcp"
  }

  dynamic "log_config" {
    for_each = var.firewall_enable_logging == true ? [{
      metadata = "INCLUDE_ALL_METADATA"
    }] : []

    content {
      metadata = log_config.value.metadata
    }
  }
}

/******************************************
  Mandatory firewall rules
 *****************************************/

resource "google_compute_firewall" "deny_all_egress" {
  destination_ranges = ["0.0.0.0/0"]
  direction          = "EGRESS"
  name               = "fw-${var.environment_code}-shared-base-65530-e-d-all-all-all"
  network            = module.main.network_name
  priority           = 65530
  project            = var.project_id

  deny {
    protocol = "all"
  }

  dynamic "log_config" {
    for_each = var.firewall_enable_logging == true ? [{
      metadata = "INCLUDE_ALL_METADATA"
    }] : []

    content {
      metadata = log_config.value.metadata
    }
  }
}
