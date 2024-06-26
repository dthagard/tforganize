# tforganize

tforganize is a command-line interface (CLI) tool designed to help you organize your Terraform code. It provides functionality to sort and restructure your Terraform files, making them easier to read and maintain. By using tforganize, you can bring order to your Terraform modules, variables, resources, and outputs.

## Features

- **Sorting**: tforganize allows you to sort your Terraform files in a predefined order. It automatically organizes your modules, variables, resources, and outputs alphabetically, improving code readability and maintainability.
- **Formatting**: In addition to sorting, tforganize also supports formatting your Terraform code. It applies consistent indentation, spacing, and line breaks, making your code more aesthetically pleasing and conforming to best practices.

### Configuration Options

tforganize offers a range of configuration options to customize its behavior according to your preferences. You can define your own sorting rules, exclude certain files or directories from sorting, and specify the desired indentation style.

## Installation

To install tforganize, follow these steps:

1. Ensure you have Golang 1.2.0 or higher installed on your system.
1. Use go to install tforganize:

```shell
go install github.com/dthagard/tforganize
```

## Usage

tforganize can be used from the command line by executing the tforganize command followed by the path to the directory containing your Terraform files. Here's the basic syntax:

```shell
Sort reads a Terraform file or folder and sorts the resources found alphabetically ascending by resource type and name.

Usage:
   sort <file | folder> [flags]

Examples:
tforganize sort main.tf

Flags:
  -g, --group-by-type           organize the resources by type in the output files
  -e, --has-header              the input files have a header
  -p, --header-pattern string   the header pattern to find the header in the input files
  -h, --help                    help for sort
  -i, --inline                  sort the resources in the input file(s) in place
  -k, --keep-header             keep the header matched in the header pattern in the output files
  -o, --output-dir string       output the results to a specific folder
  -r, --remove-comments         remove comments in the sorted file(s)

Global Flags:
      --config string   config file (default is $HOME/.tforganize.yaml)
  -d, --debug           verbose logging
```

### Running the binary directly

Sort all Terraform files in the current directory:

```shell
tforganize sort -i .
```

Sort all Terraform files in a specific directory:

```shell
tforganize sort -i /path/to/terraform/files
```

### Using docker

tforganize can be run in a container with the .tf files mounted inside of the volume:

```shell
docker run --rm -v "$(pwd):/tforganize" -w /tforganize ghcr.io/dthagard/tforganize/tforganize:latest sort -i .
```

### Using GitLab Runners

To use tforganize in a GitLab Runner, configure your `.gitlab-ci.yml` with the following:

```yaml
tforganize:
  before_script: []
  image:
    entrypoint: [""]
    name: ghcr.io/dthagard/tforganize/tforganize:latest
  rules:
    # Always run tflint on merge request events to the default branch
    - if: $CI_PIPELINE_SOURCE == "merge_request_event" && $CI_MERGE_REQUEST_TARGET_BRANCH_NAME == $CI_DEFAULT_BRANCH
      when: always
  script:
    - tforganize -i $TF_ROOT_DIRECTORY # Run the tforganize command aginst the Terraform directory
    - git diff-index --quiet HEAD -- || exit 1 # Fail the job if any changes are detected
  stage: lint
  variables:
    TERRAFORM_ROOT_DIRECTORY: <path_to_terraform_files>
```

### Using Makefile

To use tforganize in make, configure your `Makefile` with the following:

```shell
TF_FOLDERS := $(shell find . -type d -not -name '.terraform')

# tforganize will organize the terraform files in the project
.PHONY: tforganize-all
tforganize-all:
    @for dir in $(TF_FOLDERS); do \
        make tforganize dir=$$dir; \
    done;

.PHONY: tforganize
tforganize:
    @echo "Organizing $$dir...\n"; \
    docker run --rm -v $(shell pwd)/$$dir:/tforganize -w /tforganize ghcr.io/dthagard/tforganize/tforganize:latest sort -i .
```

Adjust the `find` command as needed for you specific repository configuration.

## Configuration

tforganize allows you to customize its behavior by providing a configuration file in YAML format. The default configuration file is .tforganize.yaml in the user's home directory, but you can specify a different file using the --config option.

Here is an example configuration file:

```yaml
header-pattern: |
 /**
  * Copyright 2022 Google LLC
  *
  * Licensed under the Apache License, Version 2.0 (the "License");
  * you may not use this file except in compliance with the License.
  * You may obtain a copy of the License at
  *
  *      http://www.apache.org/licenses/LICENSE-2.0
  *
  * Unless required by applicable law or agreed to in writing, software
  * distributed under the License is distributed on an "AS IS" BASIS,
  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  * See the License for the specific language governing permissions and
  * limitations under the License.
  */
has-header: true
keep-header: true
```

In this example configuration, tforganize will sort the Terraform files and keep any comments as well as prepend a header to every file.

## License

tforganize is released under the MIT License. Feel free to modify and distribute it according to your needs.

## Contributing

Contributions to tforganize are welcome! If you find a bug, have a feature request, or want to contribute code improvements, please open an issue or submit a pull request on the GitHub repository.

## Credits

tforganize is developed and maintained by @dthagard. It was inspired by the need for organizing complex Terraform projects efficiently.

## Contact

If you have any questions, suggestions, or feedback regarding tforganize, you can reach out to the project maintainer at 1454296+dthagard@users.noreply.github.com.

Happy organizing with tforganize!