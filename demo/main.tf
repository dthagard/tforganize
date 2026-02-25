output "instance_ip" {
  value = aws_instance.web.public_ip
}

output "vpc_id" {
  value = aws_vpc.main.id
}

resource "aws_instance" "web" {
  ami           = var.ami_id
  instance_type = var.instance_type
}

resource "aws_security_group" "web" {
  name   = "web-sg"
  vpc_id = aws_vpc.main.id
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

variable "ami_id" {
  description = "AMI to use for the web instance"
}

variable "instance_type" {
  default = "t3.micro"
}

variable "region" {
  default = "us-east-1"
}
