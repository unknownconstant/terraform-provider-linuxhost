package provider

import (
	"context"
	"fmt"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &UserResource{}
var _ resource.ResourceWithConfigValidators = &UserResource{}
var _ resource.ResourceWithModifyPlan = &UserResource{}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	hostData *linuxhost_client.HostData
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a user on the host, i.e. an entry in /etc/passwd.s",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The username for the user, must be unique.",
			},
			"uid": schema.NumberAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Number{numberplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "The user's UID. If unspecified, this field contains the assign UID when the user is created.",
			},
			"gid": schema.NumberAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Number{numberplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "This user's primary group ID. If unspecified this field contains the GID assigned by Linux. Cannot be set when primary_group is set.",
			},
			"primary_group": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "This user's primary group name. If specified, the group must already exist. If unspecified this field contains the group name reported by Linux.",
			},
			"groups": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "A list of group names the user is a member of. It includes the user's primary group.",
			},
			"home_directory": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "The full path to the user's home directory.",
			},
			"shell": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "The full path to the user's shell.",
			},
			"hostname": schema.StringAttribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				MarkdownDescription: "The hostname the user is created on.",
			},
		},
		Version: 1,
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.hostData, _ = req.ProviderData.(*linuxhost_client.HostData)
}

func (r *UserResource) readState(ctx context.Context, data *models.UserModel, State *tfsdk.State, Diagnostics *diag.Diagnostics) {
	linuxhost_client.RefreshGroups(r.hostData)
	users, err := linuxhost_client.RefreshUsers(r.hostData)

	tflog.Debug(ctx, "Reading state for 'user'",
		map[string]interface{}{
			"username": data.Username.String(),
		})
	if err != nil {
		Diagnostics.AddError("Failed reading back users", err.Error())
	}
	for _, user := range users {
		if !user.Username.Equal(data.Username) && !user.UID.Equal(data.UID) {
			continue
		}
		Diagnostics.Append(State.Set(ctx, user)...)
		return
	}
	Diagnostics.AddError("Failed finding user in read back users", "")
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := linuxhost_client.SetUser(r.hostData.Client, &data, nil)

	if err != nil {
		resp.Diagnostics.AddError("Failed creating user", err.Error())
	}
	r.readState(ctx, &data, &resp.State, &resp.Diagnostics)

}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.hostData == nil {
		resp.Diagnostics.AddError("Missing client", "")
		return
	}
	var data models.UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	users, err := linuxhost_client.RefreshUsers(r.hostData)
	if err != nil {
		resp.Diagnostics.AddError("Failed reading back users", err.Error())
	}
	for _, user := range users {
		if !user.Username.Equal(data.Username) && !user.UID.Equal(data.UID) {
			continue
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &user)...)
		return
	}
	resp.State.RemoveResource(ctx)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *UserResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	tflog.Debug(ctx, "Modifying plan for 'user'")
	var planned models.UserModel
	conversion := req.Plan.Get(ctx, &planned)
	if planned.Username.IsNull() {
		/// Deleting
		tflog.Debug(ctx, "Plan is to delete the user, not modifying")
		return
	}
	resp.Diagnostics.Append(conversion...)
	var state models.UserModel
	conversion = req.State.Get(ctx, &state)
	isNew := state.UID.IsNull() && state.Username.IsNull()
	if !isNew {
		resp.Diagnostics.Append(conversion...)
	}

	diff, _ := req.Plan.Raw.Diff(req.State.Raw)
	if resp.Diagnostics.HasError() {
		fmt.Println("diagnostic error")
		return
	}

	exclusiveGroupModifiers := 0
	exclusiveIdentityModifiers := 0

	for _, att := range diff {
		/// If primary_group is set, then use that over gid. Don't pass both.
		if att.Path.Equal(tftypes.NewAttributePath().WithAttributeName("primary_group")) {
			if planned.PrimaryGroup.IsUnknown() {
				continue
			}
			planned.GID = types.NumberUnknown()
			planned.Groups = types.SetUnknown(types.StringType)
			exclusiveGroupModifiers++
		} else if att.Path.Equal(tftypes.NewAttributePath().WithAttributeName("gid")) {
			if planned.GID.IsUnknown() {
				continue
			}
			planned.PrimaryGroup = types.StringUnknown()
			planned.Groups = types.SetUnknown(types.StringType)
			exclusiveGroupModifiers++
		} else if att.Path.Equal(tftypes.NewAttributePath().WithAttributeName("uid")) {
			if isNew {
				continue
			}
			exclusiveIdentityModifiers++
		} else if att.Path.Equal(tftypes.NewAttributePath().WithAttributeName("username")) {
			if isNew {
				continue
			}
			exclusiveIdentityModifiers++
		}
	}
	if exclusiveGroupModifiers > 1 {
		var state models.UserModel
		req.State.Get(ctx, &state)
		resp.Diagnostics.AddError("Group overdefined", "You may only define changes for gid or primary_group_name. The specified gid may not match the gid of the specified primary_group_name")
	}
	if exclusiveIdentityModifiers > 1 {
		var state models.UserModel
		req.State.Get(ctx, &state)
		resp.Diagnostics.AddError("User identity overdefined", "You may only define changes for username or uid. Changing the username and the uid in the same step means there is no link to the original user.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Plan.Set(ctx, &planned)

}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state models.UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	/// Read Terraform plan data into the model

	err := linuxhost_client.SetUser(r.hostData.Client, &plan, state.Username.ValueStringPointer())

	if err != nil {
		resp.Diagnostics.AddError("Failed updating user", err.Error())
	}

	r.readState(ctx, &plan, &resp.State, &resp.Diagnostics)

	RB := &models.UserModel{}
	resp.State.Get(ctx, RB)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.UserModel
	/// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	err := linuxhost_client.DeleteUser(r.hostData.Client, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete user", err.Error())
	}
	if resp.Diagnostics.HasError() {
		return
	}
}
func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *UserResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("gid"),
			path.MatchRoot("primary_group"),
		),
	}
}
