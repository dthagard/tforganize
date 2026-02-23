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

resource "aws_instance" "for_each_example" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
  provider      = aws.west
  for_each      = toset(["a", "b"])
}

resource "aws_instance" "count_example" {
  ami           = "ami-12345678"
  instance_type = "t3.micro"
  provider      = aws.west
  count         = 2
}

module "for_each_example" {
  for_each  = toset(["x", "y"])
  providers = { aws = aws.west }
  version   = "~> 2.0"
  source    = "terraform-aws-modules/vpc/aws"
}

module "count_example" {
  count     = 1
  providers = { aws = aws.west }
  version   = "~> 2.0"
  source    = "terraform-aws-modules/vpc/aws"
}
