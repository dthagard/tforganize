import {
  to = aws_instance.legacy
  id = "i-0123456789abcdef0"
}

import {
  to       = aws_s3_bucket.archive
  id       = "my-archive-bucket"
  provider = aws.west
}

check "api_health" {
  assert {
    condition     = true
    error_message = "API health check failed"
  }
}

check "db_health" {
  data "http" "db_check" {
    url = "http://localhost:5432/health"
  }

  assert {
    condition     = data.http.db_check.status_code == 200
    error_message = "DB health check failed"
  }
}
