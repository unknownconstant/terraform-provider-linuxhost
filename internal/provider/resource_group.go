package provider

import (
	"context"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &GroupResource{}

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

type GroupResource struct {
	hostData *linuxhost_client.HostData
}

func (r *GroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A group",
		Version:             1,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"gid": schema.NumberAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Number{numberplanmodifier.UseStateForUnknown()},
			},
			"members": schema.SetAttribute{
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *GroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.hostData, _ = req.ProviderData.(*linuxhost_client.HostData)
}

func (r *GroupResource) makeStateRefresher(ctx context.Context, State *tfsdk.State, Diagnostics *diag.Diagnostics) *ReadableResource[models.GroupModel] {
	tflog.Debug(ctx, "Refreshing groups")
	currentState, err := linuxhost_client.RefreshGroups(r.hostData)
	if err != nil {
		Diagnostics.AddError("Failed to refresh groups", err.Error())
		return nil
	}
	return &ReadableResource[models.GroupModel]{
		Ctx:          ctx,
		State:        State,
		Diagnostics:  Diagnostics,
		currentState: currentState,
		Equal: func(A, B *models.GroupModel) bool {
			return A.GID.Equal(B.GID) || A.Name.Equal(B.Name)
		},
		New: func(current, target *models.GroupModel) models.GroupModel { return *current },
	}
}

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	linuxhost_client.SetGroup(r.hostData.Client, &data, nil)

	r.makeStateRefresher(ctx, &resp.State, &resp.Diagnostics).InState(data, "present")
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.hostData == nil {
		resp.Diagnostics.AddError("Missing client", "")
		return
	}
	var data models.GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	r.makeStateRefresher(ctx, &resp.State, &resp.Diagnostics).InState(data, "any")

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.GroupModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state models.GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	linuxhost_client.SetGroup(r.hostData.Client, &plan, state.Name.ValueStringPointer())
	r.makeStateRefresher(ctx, &resp.State, &resp.Diagnostics).InState(plan, "present")
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.GroupModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	linuxhost_client.DeleteGroup(r.hostData.Client, &data)
	r.makeStateRefresher(ctx, &resp.State, &resp.Diagnostics).InState(data, "absent")
}
func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
