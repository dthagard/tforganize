/**
 * Copyright (c) 2025 Example Corp
 **/

resource "aws_s3_bucket" "alpha" {
  bucket = "my-alpha-bucket"
}

# This comment belongs to the beta resource
resource "aws_s3_bucket" "beta" {
  bucket = "my-beta-bucket"
}
