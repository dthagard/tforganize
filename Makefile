# Static variables
BIN_DIR := ./bin
APP_NAME := tforganize
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/dthagard/tforganize/internal/info.AppVersion=$(VERSION)

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOINSTALL = $(GOCMD) install
GOTEST = $(GOCMD) test
GOTOOL = $(GOCMD) tool

# Default target
default: all

#####################
# Phony targets
#####################

# Build the application
.PHONY: all
all: build

# Build target
.PHONY: build
build:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/${APP_NAME}

# Clean target
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

# Configure the development environment
.PHONY: configure
configure: dep
	$(GOCMD) mod verify

# Cache the dependencies locally
.PHONY: dep
dep:
	$(GOCMD) mod download

# Install target
.PHONY: install
install:
	$(GOINSTALL)

# Run target
.PHONY: run
run: build
	$(BIN_DIR)/$(APP_NAME) $(TARGET)

# Test target
.PHONY: test
test:
	$(GOTEST) -v ./...

# Lint target
.PHONY: lint
lint:
	golangci-lint run ./...

# Require every first-party statement to be covered, including the entry point.
.PHONY: test_coverage
test_coverage:
	$(GOTEST) ./... -coverpkg=./... -coverprofile=coverage.out
	$(GOTOOL) cover -func=coverage.out
	@awk 'NR > 1 { statements[$$1] = $$2; hits[$$1] += $$3 } \
		END { \
			for (block in statements) { \
				total += statements[block]; \
				if (statements[block] > 0 && hits[block] == 0) { \
					print "Uncovered block: " block; uncovered += statements[block]; \
				} \
			} \
			if (total == 0 || uncovered > 0) { \
				printf "Coverage gate failed: %d of %d statements uncovered\n", uncovered, total; exit 1; \
			} \
			printf "Coverage gate passed: all %d statements covered\n", total; \
		}' coverage.out

# Install the optional file watcher separately from normal builds.
.PHONY: configure_watch
configure_watch:
	$(GOINSTALL) github.com/githubnemo/CompileDaemon@v1.4.0

# Watch the target files and rebuild on change
.PHONY: watch
watch:
	CompileDaemon \
		-build="make build" \
		-command="make test" \
		-directory=. \
		-exclude-dir=.git \
		-exclude-dir=vendor \
		-include=Makefile \
		&
