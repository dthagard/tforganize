resource "aws_instance" "web" {
  depends_on = [aws_iam_role.web]

  lifecycle {
    create_before_destroy = true
  }

  provisioner "remote-exec" {
    inline = ["echo first"]
  }

  ebs_block_device {
    device_name = "/dev/sdh"
  }

  instance_type = "t3.micro"

  provisioner "local-exec" {
    command = "echo second"
  }

  ami      = "ami-12345678"
  for_each = toset(["a", "b"])
}

resource "terraform_data" "replacement" {
  triggers_replace = [aws_instance.web]
  input            = "value"
}

data "aws_ami" "ubuntu" {
  lifecycle {
    postcondition {
      condition     = self.architecture == "x86_64"
      error_message = "AMI must be x86_64."
    }
  }

  depends_on = [aws_iam_role.web]

  filter {
    name   = "name"
    values = ["ubuntu/*"]
  }

  owners = ["099720109477"]
  count  = 1
}

output "web_ip" {
  depends_on = [aws_instance.web]

  precondition {
    condition     = length(aws_instance.web) > 0
    error_message = "No instances."
  }

  sensitive   = true
  value       = aws_instance.web["a"].private_ip
  description = "Private IP of the web instance"
  type        = string
}
