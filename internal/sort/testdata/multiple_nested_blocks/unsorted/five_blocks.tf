resource "example" "five" {
  nested_block {
    value = "echo"
    index = 5
  }

  nested_block {
    value = "bravo"
    index = 2
  }

  nested_block {
    value = "delta"
    index = 4
  }

  nested_block {
    value = "alpha"
    index = 1
  }

  nested_block {
    value = "charlie"
    index = 3
  }
}
