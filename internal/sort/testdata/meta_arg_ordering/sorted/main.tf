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

variable "example" {
  description = "All five pre-meta args in reverse order"
  type        = string
  default     = "hello"
  nullable    = false
  sensitive   = true
}

variable "partial" {
  description = "Only some pre-meta args present"
  type        = number
  default     = 42
}
