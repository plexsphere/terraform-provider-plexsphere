// Copyright (c) plexsphere contributors
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/plexsphere/terraform-provider-plexsphere/internal/provider"
)

// version is overridden at release time via -ldflags "-X main.version=...".
var version = "dev"

// Regenerate schema + model code from the vendored OpenAPI spec.
// Run with: go generate ./...
//go:generate go run github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi generate --config generator_config.yml --output provider-code-spec.json spec/plexsphere-v1.yaml
//go:generate go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework generate resources --input provider-code-spec.json --output internal/provider --package provider
//go:generate go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework generate data-sources --input provider-code-spec.json --output internal/datasources --package datasources

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/plexsphere/plexsphere",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
