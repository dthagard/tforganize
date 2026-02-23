resource "example" "five" {
  nested_block {
    index = 5
    value = "echo"
  }

  nested_block {
    index = 2
    value = "bravo"
  }

  nested_block {
    index = 4
    value = "delta"
  }

  nested_block {
    index = 1
    value = "alpha"
  }

  nested_block {
    index = 3
    value = "charlie"
  }
}
