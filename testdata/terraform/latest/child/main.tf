import {
  id = "fixture-import-id"
  to = terraform_data.imported
}

resource "terraform_data" "imported" {
  input = "imported inside a child module"
}
