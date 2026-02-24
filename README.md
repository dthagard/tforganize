# tforganize

> Opinionated Terraform file organizer — keep modules, variables, resources, and meta-arguments in a predictable order.

`tforganize` is a CLI that rewrites `.tf` files so they match a consistent layout. It sorts blocks, enforces Terraform's canonical meta-argument order, optionally splits output by block type, and protects custom headers/comments when you want to keep them.

## Features at a glance

- **Deterministic sorting** – resources, modules, variables, outputs, checks, imports, etc. are emitted in a stable order.
- **Terraform-aware meta args** – `count`, `for_each`, `providers`, `moved`, `removed`, `check`, and friends are placed exactly where Terraform expects them.
- **Group-by-type output** – `tforganize sort -g` rewrites files into logical targets (`variables.tf`, `outputs.tf`, `checks.tf`, `imports.tf`, `main.tf`, …).
- **Header/comment control** – strip comments entirely, preserve them, or keep/apply a custom header banner.
- **Inline or out-of-place** – update files in place (`--inline`) or emit to an output directory for review/CI.
- **Configurable** – every flag has a YAML counterpart so you can save defaults in `.tforganize.yaml` or supply `--config`.
- **CI friendly** – published as a Go binary and as `ghcr.io/dthagard/tforganize/tforganize:latest` for Docker/GitLab/GitHub runners.

## Installation

> Requires Go **1.20+** (or use the Docker image below).

```bash
go install github.com/dthagard/tforganize@latest
```

Or run straight from the container registry:

```bash
docker run --rm -v "$(pwd)":/tforganize -w /tforganize ghcr.io/dthagard/tforganize/tforganize:latest sort -i .
```

## Quick start

Sort everything in the current directory in-place:

```bash
tforganize sort -i .
```

Split blocks by type (creates `variables.tf`, `outputs.tf`, `checks.tf`, `imports.tf`, etc.):

```bash
tforganize sort --group-by-type --output-dir ./sorted
```

Keep a copyright header while stripping other comments:

```bash
tforganize sort \
  --inline \
  --has-header \
  --header-pattern "$(cat header.txt)" \
  --keep-header \
  --remove-comments
```

## CLI reference (trimmed)

```text
Usage: tforganize sort <file | folder> [flags]

Flags:
  -g, --group-by-type           write each block type to its default file (see table below)
  -i, --inline                  rewrite files in place (otherwise write to --output-dir)
  -o, --output-dir string       directory for sorted files (required unless --inline)
  -r, --remove-comments         drop all comments except headers kept via --keep-header
  -e, --has-header              treat files as having a header matched by --header-pattern
  -p, --header-pattern string   regex or multi-line string that matches the header block
  -k, --keep-header             preserve the matched header in the output (requires --has-header and pattern)
      --config string           YAML config path (default $HOME/.tforganize.yaml)
  -d, --debug                   enable verbose logging
```

## Group-by-type target files

When `--group-by-type` (or `group-by-type: true` in config) is enabled, blocks are emitted to the following defaults:

| Block type  | File name        |
|-------------|------------------|
| `data`      | `data.tf`        |
| `locals`    | `locals.tf`      |
| `output`    | `outputs.tf`     |
| `terraform` | `versions.tf`    |
| `variable`  | `variables.tf`   |
| `check`     | `checks.tf`      |
| `import`    | `imports.tf`     |
| `moved`     | `main.tf`        |
| `removed`   | `main.tf`        |
| everything else | `main.tf`   |

You can feed multiple files and directories; `tforganize` builds the combined AST, sorts it, and then writes these grouped files to the chosen output.

## Configuration file

All flags can be set via YAML (default `$HOME/.tforganize.yaml` or pass `--config`). Example:

```yaml
# ~/.tforganize.yaml
group-by-type: true
inline: true
remove-comments: false
has-header: true
keep-header: true
header-pattern: |
  /**
   * Company Confidential
   */
```

Key fields:

| Key              | Description                                  |
|------------------|----------------------------------------------|
| `group-by-type`  | Same as `--group-by-type`                    |
| `inline`         | Same as `--inline`                           |
| `output-dir`     | Same as `--output-dir`                       |
| `remove-comments`| Same as `--remove-comments`                  |
| `has-header`     | Indicates a header block exists              |
| `header-pattern` | Multi-line string or regex used to match header |
| `keep-header`    | Re-emit the matched header (requires the two options above) |

`tforganize` refuses to run with `keep-header: true` unless `has-header` is true **and** `header-pattern` is non-empty — the same validation applies to CLI flags.

## Automation examples

### GitLab CI

```yaml
stages: [lint]

terraform:lint:
  stage: lint
  image:
    name: ghcr.io/dthagard/tforganize/tforganize:latest
    entrypoint: [""]
  script:
    - tforganize sort --inline "$TF_ROOT"  # reorganize files
    - git diff --quiet || {
        echo "tforganize found drift"; exit 1;
      }
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  variables:
    TF_ROOT: infrastructure
```

### Makefile loop

```make
TF_DIRS := $(shell find . -type d -not -path '*/.terraform/*')

tforganize-all:
	@for dir in $(TF_DIRS); do \
	  echo "Organizing $$dir"; \
	  tforganize sort --inline $$dir; \
	done
```

### Docker one-liner

```bash
docker run --rm -v "$(pwd)":/tforganize -w /tforganize \
  ghcr.io/dthagard/tforganize/tforganize:latest sort -i .
```

## Contributing & support

- Issues / ideas → [GitHub Issues](https://github.com/dthagard/tforganize/issues)
- PRs welcome — please run `go test ./...` and include a short description of the behavior change.
- Licensed under MIT.

Happy organizing!
