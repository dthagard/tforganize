output "policies" {
  value = {
    encoded = jsondecode(local.encoded)
    heredoc = jsondecode(local.heredoc)
    quoted  = jsondecode(local.quoted)
  }
}

locals {
  # === Policy documents ===
  encoded = jsonencode({
    Version = "2012-10-17"
    # A real HCL comment, not policy content.
    Statement = [
      {
        Sid    = "Unicode世界"
        Effect = "Allow"
        // Preserve the object-entry newline when removing this comment.
        Action = ["s3:GetObject", "s3:ListBucket"]
        Resource = [
          for/* separate keyword */bucket in ["audit", "logs"] :
          "arn:aws:s3:::${bucket}/$${aws:username}/*"
        ]
        Condition = {
          "ForAllValues:StringEquals" = {
            "aws:TagKeys" = ["environment", "owner"]
          }
          Bool = { "aws:SecureTransport" = /* a multiline
            comment is whitespace */ "true" }
          StringLike = {
            "s3:prefix" = ["home/$${aws:username}/*", "世界/é/*"]
            "example:escaped" = "quote: \" slash: \\ newline: \n"
          }
        }
      },
      {
        Effect = "Deny"
        NotAction = ["iam:Get*", "sts:GetCallerIdentity"]
        Resource = "*"
        Condition = {
          StringEquals = {
            "example:multiline" = <<-VALUE
              first

              # literal hash
              // literal slashes
              /* literal block */
              # === literal section ===
              last
            VALUE
            "example:block-shaped" = <<VALUE
payload {
}
VALUE
            "example:template" = <<-VALUE
              %{ for name in ["audit", "logs"] ~}
              ${name}/$${aws:username}
              %{ endfor ~}
            VALUE
          }
        }
      }
    ]
  })


  heredoc = <<-POLICY
    {
      "Version": "2012-10-17",

      "Statement": [
        {
          "Effect": "Allow",
          "Action": ["sts:AssumeRole"],
          "Principal": {
            "AWS": ["arn:aws:iam::123456789012:root"],
            "Service": "ec2.amazonaws.com"
          },
          "Condition": {
            "StringEquals": {
              "sts:ExternalId": "quote: \" slash: \\ Unicode: 世界",
              "aws:PrincipalTag/name": "$${aws:username}"
            },
            "StringLike": {"example:url": "https://example.test/path#fragment"}
          }
        },
        {
          "Effect": "Deny",
          "Action": "*",
          "Resource": "*",
          "Condition": {"Bool": {"aws:SecureTransport": "false"}}
        }
      ]
    }
  POLICY

  quoted = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":\"s3:GetObject\",\"Resource\":\"arn:aws:s3:::example/$${aws:username}/*\"}]}"
}
