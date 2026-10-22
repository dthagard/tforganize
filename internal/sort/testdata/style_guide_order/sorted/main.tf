data "aws_ami" "ubuntu" {
  count = 1

  owners = ["099720109477"]

  filter {
    name   = "name"
    values = ["ubuntu/*"]
  }

  lifecycle {
    postcondition {
      condition     = self.architecture == "x86_64"
      error_message = "AMI must be x86_64."
    }
  }

  depends_on = [aws_iam_role.web]
}

resource "aws_instance" "web" {
  for_each = toset(["a", "b"])

  ami           = "ami-12345678"
  instance_type = "t3.micro"

  ebs_block_device {
    device_name = "/dev/sdh"
  }

  provisioner "remote-exec" {
    inline = ["echo first"]
  }

  provisioner "local-exec" {
    command = "echo second"
  }

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [aws_iam_role.web]
}

resource "terraform_data" "replacement" {
  input            = "value"
  triggers_replace = [aws_instance.web]
}

output "web_ip" {
  type        = string
  description = "Private IP of the web instance"
  value       = aws_instance.web["a"].private_ip
  sensitive   = true

  precondition {
    condition     = length(aws_instance.web) > 0
    error_message = "No instances."
  }

  depends_on = [aws_instance.web]
}
