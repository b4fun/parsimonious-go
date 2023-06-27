#!/usr/bin/env bash

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Disable this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Development

.PHONY: test
test: fmt vet test-src test-samples ## Run all tests

.PHONY: test-src
test-src:
	go test -v ./...

.PHONY: test-samples
test-samples:
	$(call run-in-folder, samples, go test -v ./...)

.PHONY: lint
lint: ## Lint go code
	golangci-lint -v run ./...

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...	
	$(call run-in-folder, samples, go fmt ./...)

.PHONY: vet
vet: ## Run go vet
	go vet ./...
	$(call run-in-folder, samples, go vet ./...)


define run-in-folder
[ -d $(1) ] && { \
	echo "Running $(2) in $(1)"; \
	cd $(1); \
	$(2); \
	cd ..; \
}
endef