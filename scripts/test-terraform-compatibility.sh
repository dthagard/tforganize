#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
terraform="${TERRAFORM:-terraform}"
organizer="${TFORGANIZE:-$root/bin/tforganize}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
export TF_IN_AUTOMATION=1 CHECKPOINT_DISABLE=1
# Keep developer configuration from changing the compatibility scenario.
export HOME="$work/home"
mkdir -p "$HOME"
printf '{}\n' > "$work/config.yaml"

version="$("$terraform" version -json | jq -r .terraform_version)"
fixtures=(modern)
case "$version" in
  1.15.*) ;;
  *) fixtures+=(latest) ;;
esac

for fixture in "${fixtures[@]}"; do
  dir="$work/$fixture"
  cp -R "$root/testdata/terraform/$fixture" "$dir"
  (
    cd "$dir"
    "$terraform" init -backend=false -input=false -no-color
    "$terraform" validate -no-color
    "$terraform" plan -input=false -refresh=false -lock=false -out=before.plan -no-color
    "$terraform" show -json before.plan | jq -S '{planned_values, resource_changes, output_changes}' > before.json
    "$organizer" --config "$work/config.yaml" sort --recursive --inline .
    "$organizer" --config "$work/config.yaml" sort --recursive --check .
    "$terraform" validate -no-color
    "$terraform" plan -input=false -refresh=false -lock=false -out=after.plan -no-color
    "$terraform" show -json after.plan | jq -S '{planned_values, resource_changes, output_changes}' > after.json
    diff -u before.json after.json
    # Grouping must preserve the same Terraform module semantics.
    "$organizer" --config "$work/config.yaml" sort --recursive --group-by-type --output-dir "$work/$fixture-grouped" .
    cd "$work/$fixture-grouped"
    "$terraform" init -backend=false -input=false -no-color
    "$terraform" validate -no-color
    "$terraform" plan -input=false -refresh=false -lock=false -out=grouped.plan -no-color
    "$terraform" show -json grouped.plan | jq -S '{planned_values, resource_changes, output_changes}' > grouped.json
    diff -u "$dir/before.json" grouped.json
  )
done

# fmt parses provider-dependent syntax without installing or invoking providers.
cp -R "$root/testdata/terraform/syntax" "$work/syntax"
"$terraform" fmt -write=false "$work/syntax"
"$organizer" --config "$work/config.yaml" sort --inline "$work/syntax/main.tf"
"$organizer" --config "$work/config.yaml" sort --check "$work/syntax/main.tf"
"$terraform" fmt -write=false "$work/syntax"
printf 'Terraform %s: validation, plan equivalence, grouping, and syntax checks passed.\n' "$version"
