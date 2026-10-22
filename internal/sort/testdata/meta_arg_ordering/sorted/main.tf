variable "example" {
  type        = string
  description = "All five pre-meta args in reverse order"
  default     = "hello"
  sensitive   = true
  nullable    = false
}

variable "partial" {
  type        = number
  description = "Only some pre-meta args present"
  default     = 42
}

resource "aws_instance" "count_example" {
  count    = 2
  provider = aws.west

  ami           = "ami-12345678"
  instance_type = "t3.micro"
}

resource "aws_instance" "for_each_example" {
  for_each = toset(["a", "b"])
  provider = aws.west

  ami           = "ami-12345678"
  instance_type = "t3.micro"
}

module "count_example" {
  source    = "terraform-aws-modules/vpc/aws"
  version   = "~> 2.0"
  providers = { aws = aws.west }
  count     = 1
}

module "for_each_example" {
  source    = "terraform-aws-modules/vpc/aws"
  version   = "~> 2.0"
  providers = { aws = aws.west }
  for_each  = toset(["x", "y"])
}
