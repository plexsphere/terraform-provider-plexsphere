SPEC_URL ?= https://api.plexsphere.com/plexsphere-v1.yaml
SPEC     := spec/plexsphere-v1.yaml
GOBIN    := $(shell go env GOPATH)/bin

.PHONY: help tools fetch-spec generate generate-openapi generate-framework build install test fmt

help:
	@echo "Targets:"
	@echo "  tools        Install the pinned codegen tools into GOBIN"
	@echo "  fetch-spec   Download + checksum the OpenAPI spec into spec/"
	@echo "  generate     Regenerate schema/model code from the spec (openapi -> framework)"
	@echo "  build        Build the provider binary"
	@echo "  install      go install the provider"
	@echo "  test         Run unit tests"
	@echo "  fmt          gofmt -s the tree"

tools:
	go install github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi
	go install github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework

fetch-spec:
	curl -fsSL "$(SPEC_URL)" -o "$(SPEC)"
	shasum -a 256 "$(SPEC)" | awk '{print $$1}' > "$(SPEC).sha256"
	@echo "spec updated: $$(wc -c < $(SPEC)) bytes, sha256 $$(cat $(SPEC).sha256)"

# Full regeneration pipeline. Equivalent to `go generate ./...`.
generate: generate-openapi generate-framework fmt

generate-openapi:
	$(GOBIN)/tfplugingen-openapi generate \
		--config generator_config.yml \
		--output provider-code-spec.json \
		$(SPEC)

generate-framework:
	$(GOBIN)/tfplugingen-framework generate resources \
		--input provider-code-spec.json --output internal/provider --package provider
	$(GOBIN)/tfplugingen-framework generate data-sources \
		--input provider-code-spec.json --output internal/datasources --package datasources

build:
	go build -o terraform-provider-plexsphere .

install:
	go install .

test:
	go test ./...

fmt:
	gofmt -w -s .
