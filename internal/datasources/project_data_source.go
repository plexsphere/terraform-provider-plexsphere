// Copyright (c) plexsphere contributors
// SPDX-License-Identifier: BUSL-1.1

package datasources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	plexsphere "github.com/plexsphere/plexsphere-sdk-go"
)

var (
	_ datasource.DataSource              = (*projectDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*projectDataSource)(nil)
)

// NewProjectDataSource is the constructor registered with the provider.
func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

type projectDataSource struct {
	client *plexsphere.APIClient
}

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = ProjectDataSourceSchema(ctx)
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*plexsphere.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *plexsphere.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project, httpResp, err := d.client.TenancyAPI.GetProject(ctx, data.Id.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project", formatAPIError(err, httpResp))
		return
	}

	data.DomainId = types.StringValue(project.DomainId)
	data.Name = types.StringValue(project.Name)
	data.Slug = types.StringValue(project.Slug)
	data.Description = types.StringPointerValue(project.Description)
	data.SubRangeCidr = types.StringPointerValue(project.SubRangeCidr)
	data.CreatedAt = types.StringValue(project.CreatedAt.Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(project.UpdatedAt.Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// formatAPIError unwraps the SDK's GenericOpenAPIError so the
// application/problem+json body is surfaced alongside the HTTP status.
func formatAPIError(err error, resp *http.Response) string {
	msg := err.Error()

	var genPtr *plexsphere.GenericOpenAPIError
	var genVal plexsphere.GenericOpenAPIError
	var body []byte
	switch {
	case errors.As(err, &genPtr):
		body = genPtr.Body()
	case errors.As(err, &genVal):
		body = genVal.Body()
	}

	if resp != nil {
		msg = fmt.Sprintf("%s (HTTP %d)", msg, resp.StatusCode)
	}
	if len(body) > 0 {
		msg = fmt.Sprintf("%s: %s", msg, string(body))
	}
	return msg
}
