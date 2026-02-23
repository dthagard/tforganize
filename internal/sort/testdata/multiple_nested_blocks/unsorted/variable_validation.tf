variable "instance_count" {
  description = "Number of instances to create"
  type        = number
  default     = 1

  validation {
    condition     = var.instance_count > 0
    error_message = "instance_count must be greater than 0."
  }

  validation {
    condition     = var.instance_count <= 10
    error_message = "instance_count must be 10 or fewer."
  }

  validation {
    condition     = var.instance_count != 7
    error_message = "instance_count must not be 7 (reserved)."
  }
}
