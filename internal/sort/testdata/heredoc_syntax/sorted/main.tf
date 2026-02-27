resource "aws_iam_policy" "admin" {
  name   = "admin"
  policy = <<-EOF
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Action": "*",
          "Resource": "*"
        }
      ]
    }
  EOF
}

resource "aws_iam_policy" "example" {
  name   = "example"
  policy = <<-EOF
    {
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Action": "s3:GetObject",
          "Resource": "*"
        }
      ]
    }
  EOF
}
