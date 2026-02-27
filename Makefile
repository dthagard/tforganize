# Static variables
BIN_DIR := ./bin
APP_NAME := tforganize
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/dthagard/tforganize/internal/info.AppVersion=$(VERSION)

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOCOMPILE = $(GOTOOL) compile
GOGET = $(GOCMD) get
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
all: configure build

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
configure:
	$(GOCMD) mod tidy
	$(GOINSTALL) github.com/githubnemo/CompileDaemon@latest
	$(GOGET) github.com/dthagard/tforganize

# Cache the dependencies locally
.PHONY: dep
dep:
	go mod download

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

# Generate the test coverage report
.PHONY: test_coverage
test_coverage:
	$(GOTEST) ./... -coverprofile=coverage.out

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
