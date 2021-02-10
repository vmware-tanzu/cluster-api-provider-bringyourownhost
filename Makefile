# Copyright the Cluster API Provider BYOH contributors.
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# If you update this file, please follow
# https://suva.sh/posts/well-documented-makefiles

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL := /usr/bin/env bash

.DEFAULT_GOAL := help

# Use GOPROXY environment variable if set
GOPROXY := $(shell go env GOPROXY)
ifeq (,$(strip $(GOPROXY)))
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Active module mode, as we use go modules to manage dependencies
export GO111MODULE := on

# Directories
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BIN_DIR := $(ROOT_DIR)/bin
TOOLS_DIR := $(ROOT_DIR)/hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin

# Binaries.
# Need to use abspath so we can invoke these from subdirectories
GOLANGCI_LINT := $(abspath $(TOOLS_BIN_DIR)/golangci-lint)
CONTROLLER_GEN := $(abspath $(TOOLS_BIN_DIR)/controller-gen)
CONVERSION_GEN := $(abspath $(TOOLS_BIN_DIR)/conversion-gen)

# Architecture variables
ARCH ?= amd64
ALL_ARCH = amd64

# Common docker variables
IMAGE_NAME ?= manager
PULL_POLICY ?= Always

# Release docker variables
RELEASE_REGISTRY := gcr.io/cluster-api-provider-byoh/release
RELEASE_CONTROLLER_IMG := $(RELEASE_REGISTRY)/$(IMAGE_NAME)

# Development Docker variables
DEV_REGISTRY ?= gcr.io/$(shell gcloud config get-value project)
DEV_CONTROLLER_IMG ?= $(DEV_REGISTRY)/vsphere-$(IMAGE_NAME)
DEV_TAG ?= dev
DEV_MANIFEST_IMG := $(DEV_CONTROLLER_IMG)-$(ARCH)

# Hosts running SELinux need :z added to volume mounts
SELINUX_ENABLED := $(shell cat /sys/fs/selinux/enforce 2> /dev/null || echo 0)

ifeq ($(SELINUX_ENABLED),1)
  DOCKER_VOL_OPTS?=:z
endif

# Set build time variables including git version details
LDFLAGS := $(shell hack/version.sh)

## --------------------------------------
## Help
## --------------------------------------

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(CONTROLLER_GEN): $(TOOLS_DIR)/go.mod # Build controller-gen from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(TOOLS_BIN_DIR)/controller-gen sigs.k8s.io/controller-tools/cmd/controller-gen

$(CONVERSION_GEN): $(TOOLS_DIR)/go.mod
	cd $(TOOLS_DIR); go build -tags=tools -o $(TOOLS_BIN_DIR)/conversion-gen k8s.io/code-generator/cmd/conversion-gen

$(GOLANGCI_LINT): $(TOOLS_DIR)/go.mod # Build golangci-lint from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(TOOLS_BIN_DIR)/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint

## --------------------------------------
## Generate
## --------------------------------------

.PHONY: modules
modules: ## Run go mod to ensure modules are up to date.
	go mod tidy
	cd $(TOOLS_DIR); go mod tidy

.PHONY: generate
generate: ## Generate code
	$(MAKE) generate-go
	$(MAKE) generate-manifests

.PHONY: generate-go
generate-go: $(CONTROLLER_GEN) ## Runs Go related generate targets
	go generate ./...
	$(CONTROLLER_GEN) \
		paths=./api/... \
		object:headerFile=./hack/boilerplate/boilerplate.generatego.txt

.PHONY: generate-manifests
generate-manifests: $(CONTROLLER_GEN) ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) \
		paths=./api/... \
		paths=./controllers/... \
		crd:crdVersions=v1 \
		rbac:roleName=manager-role \
		output:crd:dir=./config/crd/bases \
		output:webhook:dir=./config/webhook \
		output:rbac:dir=./config/rbac \
		webhook

## --------------------------------------
## Linting 
## --------------------------------------

.PHONY: lint
lint: ## Run all the lint targets
	$(MAKE) lint-go-full

GOLANGCI_LINT_FLAGS ?= --fast=true
.PHONY: lint-go
lint-go: $(GOLANGCI_LINT) ## Lint the go codebase
	$(GOLANGCI_LINT) run -v $(GOLANGCI_LINT_FLAGS)

.PHONY: lint-go-full 
lint-go-full: ## Lint the go codebase running slower linters to detect possible issues
	$(MAKE) GOLANGCI_LINT_FLAGS="--fast=false" lint-go 

## --------------------------------------
## Check
## --------------------------------------

.PHONY: verify
verify: ## Run all the verify targets
	$(MAKE) verify-boilerplate
	$(MAKE) verify-shellcheck
	$(MAKE) verify-modules
	$(MAKE) verify-generated

.PHONY: verify-boilerplate
verify-boilerplate: ## verify boilerplate
	./hack/verify-boilerplate.sh

.PHONY: verify-modules
verify-modules: modules ## verify go modules are up to date
	@if !(git diff --quiet HEAD -- go.sum go.mod hack/tools/go.mod hack/tools/go.sum); then \
		git diff; \
		echo "go module files are out of date"; exit 1; \
	fi

.PHONY: verify-generated
verify-generated: generate ## verify generated files are up to date
	@if !(git diff --quiet HEAD); then \
		git diff; \
		echo "generated files are out of date, run make generate"; exit 1; \
	fi

.PHONY: verify-shellcheck
verify-shellcheck: ## verify shellcheck
	./hack/verify-shellcheck.sh

## --------------------------------------
## Testing
## --------------------------------------

.PHONY: test
test: ## Run tests
	go test -v ./api/... ./controllers/...

## --------------------------------------
## Binaries
## --------------------------------------

.PHONY: $(MANAGER)
manager: generate-go ## Build manager binary
	go build -o $(BIN_DIR)/manager -ldflags "$(LDFLAGS) -extldflags '-static' -w -s"

agent: generate-go ## Build agent binary
	go build -o $(BIN_DIR)/agent -ldflags "$(LDFLAGS) -extldflags '-static' -w -s" vmware-tanzu/cluster-api-provider-byoh/agent

## --------------------------------------
## Docker
## --------------------------------------

## --------------------------------------
## Release
## --------------------------------------

release-binaries: ## Builds the binaries to publish with a release
	RELEASE_BINARY=./agent GOOS=linux GOARCH=amd64 $(MAKE) release-binary
	# RELEASE_BINARY=./agent GOOS=darwin GOARCH=amd64 $(MAKE) release-binary

release-binary: $(RELEASE_DIR)
	docker run \
		--rm \
		-e CGO_ENABLED=0 \
		-e GOOS=$(GOOS) \
		-e GOARCH=$(GOARCH) \
		-v "$$(pwd):/workspace$(DOCKER_VOL_OPTS)" \
		-w /workspace \
		golang:1.15.3 \
		go build -a -ldflags "$(LDFLAGS) -extldflags '-static'" \
		-o ./bin/$(notdir $(RELEASE_BINARY))-$(GOOS)-$(GOARCH) $(RELEASE_BINARY)

## --------------------------------------
## Cleanup 
## --------------------------------------

.PHONY: clean
clean: ## Remove all generated files
	$(MAKE) clean-bin
	$(MAKE) clean-tools-bin

.PHONY: clean-bin
clean-bin: ## Remove all generated binaries
	rm -rf $(BIN_DIR)

.PHONY: clean-tools-bin
clean-tools-bin: ## Remove all generated tool binaries
	rm -rf $(TOOLS_BIN_DIR)
	