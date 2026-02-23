resource "example" "two" {
  nested_block {
    value = "second"
    label = "b"
  }

  nested_block {
    value = "first"
    label = "a"
  }
}
