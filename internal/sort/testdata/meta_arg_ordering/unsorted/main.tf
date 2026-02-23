variable "example" {
  sensitive   = true
  nullable    = false
  default     = "hello"
  type        = string
  description = "All five pre-meta args in reverse order"
}

variable "partial" {
  default     = 42
  description = "Only some pre-meta args present"
  type        = number
}

resource "aws_instance" "example" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
  provider      = aws.west
  for_each      = toset(["a", "b"])
  count         = 2
}

module "example" {
  for_each  = toset(["x", "y"])
  count     = 1
  providers = { aws = aws.west }
  version   = "~> 2.0"
  source    = "terraform-aws-modules/vpc/aws"
}
