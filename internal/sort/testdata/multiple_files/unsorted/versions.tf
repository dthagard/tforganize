terraform {
  required_version = ">= 0.13"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 3.50"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = ">= 3.50"
    }
  }

  provider_meta "google" {
    module_name = "blueprints/terraform/terraform-example-foundation:base_shared_vpc/v3.0.0"
  }

  provider_meta "google-beta" {
    module_name = "blueprints/terraform/terraform-example-foundation:base_shared_vpc/v3.0.0"
  }
}
