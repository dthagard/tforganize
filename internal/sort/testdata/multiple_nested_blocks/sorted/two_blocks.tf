resource "example" "two" {
  nested_block {
    label = "b"
    value = "second"
  }

  nested_block {
    label = "a"
    value = "first"
  }
}
