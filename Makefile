# Static variables
BIN_DIR := ./bin
APP_NAME := tfsort

# Dynamic variables
SRCS := $(wildcard *.go)
OBJS := $(patsubst %.go, $(BIN_DIR)/%.o, $(SRCS))

# User variables
TARGET := sample_hcl2json.hcl

# Go parameters
GOCMD = go
GOTOOL = $(GOCMD) tool
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOCOMPILE = $(GOTOOL) compile
GOINSTALL = $(GOCMD) install
GOTEST = $(GOCMD) test

# Default target
default: all

#####################
# Build targets
#####################

$(BIN_DIR)/$(APP_NAME): $(OBJS)
	$(GOBUILD) -o $@ $(OBJS)

$(BIN_DIR)/%.o: %.go
	$(GOCOMPILE) -o $@ $<

#####################
# Phony targets
#####################

# Build the application
.PHONY: all
all: $(BIN_DIR)/$(APP_NAME)

# Build target
.PHONY: build
build:
	$(GOBUILD) -o $(BIN_DIR)/${APP_NAME}

# Clean target
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

# Configure the development environment
.PHONY: configure
configure:
	$(GOCMD) mod tidy
	$(GOCMD) mod vendor
	$(GOINSTALL) github.com/githubnemo/CompileDaemon@latest

# Cache the dependencies locally
.PHONY: dep
dep:
	go mod download

# Install target
.PHONY: install
install:
	$(GOINSTALL)

# Run the golangci-lint tool
.PHONY: lint
lint:
	golangci-lint run --enable-all

# Run target
.PHONY: run
run: build
	$(BIN_DIR)/$(APP_NAME) $(TARGET)

# Test target
.PHONY: test
test:
	$(GOTEST) -v ./...

# Generate the test coverage report
.PHONY: test_coverage
test_coverage:
	go test ./... -coverprofile=coverage.out

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
