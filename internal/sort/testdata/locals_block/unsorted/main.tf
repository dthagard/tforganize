locals {
  tags = {
    Environment = var.environment
    Project     = var.project
  }
  name_prefix = "${var.project}-${var.environment}"
}
