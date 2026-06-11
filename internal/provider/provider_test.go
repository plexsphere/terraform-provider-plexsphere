// Copyright (c) plexsphere contributors
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestProviderSchema boots the provider through the plugin protocol and asks
// for its full schema. This validates the provider, the generated resource and
// data-source schemas, and the hand-written plan-modifier overlay all at once —
// e.g. it would fail if UseStateForUnknown were applied to a non-computed
// attribute or an attribute were both Required and Computed.
func TestProviderSchema(t *testing.T) {
	ctx := context.Background()
	server := providerserver.NewProtocol6(New("test")())()

	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema returned an error: %s", err)
	}

	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("schema diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}

	if _, ok := resp.ResourceSchemas["plexsphere_project"]; !ok {
		t.Error("expected resource plexsphere_project to be registered")
	}
	if _, ok := resp.DataSourceSchemas["plexsphere_project"]; !ok {
		t.Error("expected data source plexsphere_project to be registered")
	}
}
