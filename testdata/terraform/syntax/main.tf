# Syntax-only: provider-dependent declarations are never applied.
action "aws_lambda_invoke" "notify" {
  config {
    payload = caller.output
    function_name = "fixture"
  }
  provider = aws.alias
  count = 1
}

ephemeral "random_password" "session" {
  lifecycle {
    postcondition {
      error_message = "Password must not be empty."
      condition = length(self.result) > 0
    }
  }
  length = 16
  for_each = toset(["fixture"])
}

resource "terraform_data" "trigger" {
  lifecycle {
    action_trigger {
      on_failure = continue
      actions = [action.aws_lambda_invoke.notify[0]]
      events = [before_destroy, after_destroy]
    }
  }
  input = "fixture"
}
