#!/usr/bin/env bash
set -euo pipefail

if ! command -v tforganize &>/dev/null; then
  echo "ERROR: tforganize is not installed or not on PATH." >&2
  echo "Install it with: go install github.com/dthagard/tforganize@latest" >&2
  echo "Or use the 'tforganize-docker' hook to run via container instead." >&2
  exit 1
fi

# pre-commit passes staged filenames as positional arguments. When invoked
# outside of pre-commit (e.g. in CI or by hand) default to the current directory.
if [[ $# -eq 0 ]]; then
  tforganize sort --inline .
else
  tforganize sort --inline "$@"
fi
