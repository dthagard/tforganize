resource "example" "test" {
  nested_block {
    z_attr = "third"
    a_attr = "alpha"
  }

  nested_block {
    z_attr = "second"
    a_attr = "beta"
  }

  nested_block {
    z_attr = "first"
    a_attr = "gamma"
  }
}
