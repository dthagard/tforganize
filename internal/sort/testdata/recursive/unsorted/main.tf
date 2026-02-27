resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}

resource "aws_instance" "app" {
  ami           = "ami-67890"
  instance_type = "t2.small"
}
