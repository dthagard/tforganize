output "message" {
  value = terraform_data.example.output
  description = "Unicode and template expressions survive organization."
}

check "names" {
  assert {
    error_message = "At least one name is required."
    condition = length(var.names) > 0
  }
}

removed {
  lifecycle {
    destroy = false
  }
  from = terraform_data.retired
}

moved {
  to = terraform_data.example
  from = terraform_data.old
}

resource "terraform_data" "example" {
  lifecycle {
    precondition {
      error_message = "Names must not be empty."
      condition = alltrue([for name in var.names : length(name) > 0])
    }
  }
  input = local.message
}

locals {
  message = <<-EOT
    Hello, ${join(", ", var.names)}!
    %{ for name in var.names ~}
    ${upper(name)} — preserved
    %{ endfor ~}
  EOT
}

variable "names" {
  validation {
    error_message = "Names must be unique."
    condition = length(distinct(var.names)) == length(var.names)
  }
  default = ["世界", "Terraform"]
  type = list(string)
  description = "Names to render."
}

terraform {
  required_version = ">= 1.15.0"
}
