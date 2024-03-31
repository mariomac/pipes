.DEFAULT_GOAL := verify

TEST_OUTPUT ?= ./out

# go-install-tool will 'go install' any package $2 and install it locally to $1.
# This will prevent that they are installed in the $USER/go/bin folder and different
# projects ca have different versions of the tools
PROJECT_DIR := $(shell dirname $(abspath $(firstword $(MAKEFILE_LIST))))

# prereqs binary dependencies
TOOLS_DIR ?= $(PROJECT_DIR)/bin
GOLANGCI_LINT = $(TOOLS_DIR)/golangci-lint
GOIMPORTS_REVISER = $(TOOLS_DIR)/goimports-reviser

define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(TOOLS_DIR) GOFLAGS="-mod=mod" go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

define check_format
	$(shell $(foreach FILE, $(shell find . -name "*.go" -not -path "./vendor/*"), \
		$(GOIMPORTS_REVISER) -list-diff -output stdout $(FILE);))
endef

.PHONY: prereqs
prereqs:
	@echo "### Check if prerequisites are met, and installing missing dependencies"
	mkdir -p $(TEST_OUTPUT)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.2)
	$(call go-install-tool,$(GOIMPORTS_REVISER),github.com/incu6us/goimports-reviser/v3@v3.6.4)

.PHONY: fmt
fmt: prereqs
	@echo "### Formatting code and fixing imports"
	@$(foreach FILE, $(shell find . -name "*.go" -not -path "./vendor/*"), \
		$(GOIMPORTS_REVISER) $(FILE);)

.PHONY: checkfmt
checkfmt:
	@echo '### check correct formatting and imports'
	@if [ "$(strip $(check_format))" != "" ]; then \
		echo "$(check_format)"; \
		echo "Above files are not properly formatted. Run 'make fmt' to fix them"; \
		exit 1; \
	fi

.PHONY: lint
lint: prereqs checkfmt
	@echo "### Linting code"
	$(GOLANGCI_LINT) run ./... --timeout=6m

.PHONY: test
test:
	@echo "### Testing code"
	go test -race -mod vendor -a ./pkg/... -coverpkg=./pkg/... -coverprofile $(TEST_OUTPUT)/cover.all.txt

.PHONY: verify
verify: prereqs lint test