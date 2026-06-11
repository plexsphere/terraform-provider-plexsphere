// Copyright (c) plexsphere contributors
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	plexsphere "github.com/plexsphere/plexsphere-sdk-go"
)

var (
	_ resource.Resource                = (*projectResource)(nil)
	_ resource.ResourceWithConfigure   = (*projectResource)(nil)
	_ resource.ResourceWithImportState = (*projectResource)(nil)
)

// NewProjectResource is the constructor registered with the provider.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *plexsphere.APIClient
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema starts from the generated schema (ProjectResourceSchema lives in
// project_resource_gen.go — "DO NOT EDIT") and overlays plan-modifier
// behaviour the OpenAPI spec implies but the generator cannot express:
//
//   - id / created_at are server-assigned and stable -> UseStateForUnknown,
//     so updates don't show them as "known after apply".
//   - domain_id / slug are absent from the PATCH body (immutable) -> changing
//     them forces a replace rather than a (impossible) in-place update.
//
// Keeping this overlay here means we never edit the generated file, so it can
// be regenerated from the spec at any time.
func (r *projectResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := ProjectResourceSchema(ctx)

	for _, name := range []string{"id", "created_at"} {
		if attr, ok := s.Attributes[name].(schema.StringAttribute); ok {
			attr.PlanModifiers = append(attr.PlanModifiers, stringplanmodifier.UseStateForUnknown())
			s.Attributes[name] = attr
		}
	}

	for _, name := range []string{"domain_id", "slug"} {
		if attr, ok := s.Attributes[name].(schema.StringAttribute); ok {
			attr.PlanModifiers = append(attr.PlanModifiers, stringplanmodifier.RequiresReplace())
			s.Attributes[name] = attr
		}
	}

	resp.Schema = s
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*plexsphere.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *plexsphere.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plexsphere.NewProjectCreateRequest(
		plan.DomainId.ValueString(),
		plan.Name.ValueString(),
		plan.Slug.ValueString(),
	)
	if isSet(plan.Description) {
		body.Description = plan.Description.ValueStringPointer()
	}
	if isSet(plan.SubRangeCidr) {
		body.SubRangeCidr = plan.SubRangeCidr.ValueStringPointer()
	}

	created, httpResp, err := r.client.TenancyAPI.CreateProject(ctx).
		ProjectCreateRequest(*body).
		Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to create project", formatAPIError(err, httpResp))
		return
	}

	state := flattenProject(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, httpResp, err := r.client.TenancyAPI.GetProject(ctx, state.Id.ValueString()).Execute()
	if err != nil {
		// Gone server-side: drop from state so Terraform plans a recreate.
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read project", formatAPIError(err, httpResp))
		return
	}

	state = flattenProject(current)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only the patchable fields appear in ProjectPatchRequest; domain_id and
	// slug are immutable and handled by RequiresReplace in Schema().
	body := plexsphere.NewProjectPatchRequest()
	if isSet(plan.Name) {
		body.Name = plan.Name.ValueStringPointer()
	}
	if isSet(plan.Description) {
		body.Description = plan.Description.ValueStringPointer()
	}
	if isSet(plan.SubRangeCidr) {
		body.SubRangeCidr = plan.SubRangeCidr.ValueStringPointer()
	}

	updated, httpResp, err := r.client.TenancyAPI.PatchProject(ctx, state.Id.ValueString()).
		ProjectPatchRequest(*body).
		Execute()
	if err != nil {
		resp.Diagnostics.AddError("Unable to update project", formatAPIError(err, httpResp))
		return
	}

	newState := flattenProject(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.TenancyAPI.DeleteProject(ctx, state.Id.ValueString()).Execute()
	if err != nil {
		// Already gone counts as a successful delete.
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Unable to delete project", formatAPIError(err, httpResp))
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// flattenProject maps a plexsphere-sdk-go ProjectResponse onto the generated
// Terraform model. This is the seam where the two model worlds meet — the only
// glue the codegen pipeline deliberately leaves to be hand-written.
func flattenProject(p *plexsphere.ProjectResponse) ProjectModel {
	return ProjectModel{
		Id:           types.StringValue(p.Id),
		DomainId:     types.StringValue(p.DomainId),
		Name:         types.StringValue(p.Name),
		Slug:         types.StringValue(p.Slug),
		Description:  types.StringPointerValue(p.Description),
		SubRangeCidr: types.StringPointerValue(p.SubRangeCidr),
		CreatedAt:    types.StringValue(p.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:    types.StringValue(p.UpdatedAt.Format(time.RFC3339)),
	}
}

// isSet reports whether a string attribute carries a concrete value (neither
// null nor unknown) and may therefore be forwarded to the API.
func isSet(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown()
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
