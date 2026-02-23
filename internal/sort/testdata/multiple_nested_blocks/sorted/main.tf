resource "example" "test" {
  nested_block {
    a_attr = "alpha"
    z_attr = "third"
  }

  nested_block {
    a_attr = "beta"
    z_attr = "second"
  }

  nested_block {
    a_attr = "gamma"
    z_attr = "first"
  }
}
