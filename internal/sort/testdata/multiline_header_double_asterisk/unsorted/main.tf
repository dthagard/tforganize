/**
 * Copyright (c) 2025 Example Corp
 *
 * This file is managed by Terraform.
 **/

resource "aws_s3_bucket" "beta" {
  bucket = "my-beta-bucket"
}

resource "aws_s3_bucket" "alpha" {
  bucket = "my-alpha-bucket"
}
