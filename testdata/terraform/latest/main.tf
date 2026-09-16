output "stored" {
  value = terraform_data.example.store.sensitive_output
  sensitive = true
}

module "imported" {
  source = "./child"
}

resource "terraform_data" "example" {
  store {
    version = 1
    sensitive = true
    replace = false
    input = var.transient
  }
  lifecycle {
    destroy = false
  }
  input = "ordinary input"
}

variable "transient" {
  sensitive = true
  ephemeral = true
  default = "fixture-only-not-a-secret"
  type = string
}

terraform {
  required_version = ">= 1.16.0"
}
