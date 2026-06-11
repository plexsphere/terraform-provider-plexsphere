// Copyright (c) plexsphere contributors
// SPDX-License-Identifier: BUSL-1.1

//go:build tools

// Package tools pins the code-generation toolchain in go.mod so that
// `go generate ./...` and CI use the exact versions locked here — mirroring
// the version-pinning philosophy of plexsphere-sdk-generator.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework"
	_ "github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi"
)
