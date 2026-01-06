GOCACHE ?= $(CURDIR)/.cache/go-build

.PHONY: docs-reference docs-build docs-claims docs
# Generate reference docs from source

docs-reference:
	@mkdir -p $(GOCACHE)
	GOCACHE=$(GOCACHE) go run scripts/gen_reference.go

# Build docs with strict mode (matches CI)
docs-build:
	mkdocs build --strict

# Verify docs Claim-ID markers against the claims ledger
docs-claims:
	@mkdir -p $(GOCACHE)
	GOCACHE=$(GOCACHE) go run ./scripts/claims_check

# Generate reference docs and build the site
docs: docs-reference docs-build
