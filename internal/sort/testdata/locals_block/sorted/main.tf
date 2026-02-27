locals {
  name_prefix = "${var.project}-${var.environment}"
  tags = {
    Environment = var.environment
    Project     = var.project
  }
}
