// Copyright (c) plexsphere contributors
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	plexsphere "github.com/plexsphere/plexsphere-sdk-go"
	"github.com/plexsphere/terraform-provider-plexsphere/internal/datasources"
)

var _ provider.Provider = (*plexsphereProvider)(nil)

// New returns the provider constructor wired into main.go.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &plexsphereProvider{version: version}
	}
}

type plexsphereProvider struct {
	version string
}

type plexsphereProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func (p *plexsphereProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "plexsphere"
	resp.Version = p.version
}

func (p *plexsphereProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage plexsphere resources through the plexsphere v1 API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the plexsphere API (e.g. https://api.plexsphere.com). May also be set via the PLEXSPHERE_ENDPOINT environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer token used to authenticate against the plexsphere API. May also be set via the PLEXSPHERE_TOKEN environment variable.",
			},
		},
	}
}

func (p *plexsphereProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config plexsphereProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Explicit config wins over environment.
	endpoint := os.Getenv("PLEXSPHERE_ENDPOINT")
	if isSet(config.Endpoint) {
		endpoint = config.Endpoint.ValueString()
	}
	token := os.Getenv("PLEXSPHERE_TOKEN")
	if isSet(config.Token) {
		token = config.Token.ValueString()
	}

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Missing plexsphere API endpoint",
			"Set the provider `endpoint` argument or the PLEXSPHERE_ENDPOINT environment variable.")
	}
	if token == "" {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Missing plexsphere API token",
			"Set the provider `token` argument or the PLEXSPHERE_TOKEN environment variable.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	sdkConfig := plexsphere.NewConfiguration()
	// The spec carries no `servers` block by design; inject the endpoint here.
	sdkConfig.Servers = plexsphere.ServerConfigurations{{URL: endpoint}}
	sdkConfig.UserAgent = fmt.Sprintf("terraform-provider-plexsphere/%s", p.version)
	// Bearer auth: the spec exempts bearer-authenticated requests from the CSRF
	// handshake, so a static Authorization header is all the provider needs.
	sdkConfig.AddDefaultHeader("Authorization", "Bearer "+token)

	client := plexsphere.NewAPIClient(sdkConfig)
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *plexsphereProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
	}
}

func (p *plexsphereProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewProjectDataSource,
	}
}
