package provider

import (
	"context"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &IfBridgeResource{}

func NewIfBridgeResource() resource.Resource {
	return &IfBridgeResource{}
}

type IfBridgeResource struct {
	LinuxhostCommonResource
}

var _ IsLinuxhostIFResource = &IfBridgeResource{}

func (r *IfBridgeResource) GetHostData() *linuxhost_client.HostData {
	return r.LinuxhostCommonResource.hostData
}

func (r *IfBridgeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_if_bridge"
}

func (r *IfBridgeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := commonInterfaceSchema()
	resp.Schema = schema.Schema{
		MarkdownDescription: "A bridge interface",
		Version:             1,
		Attributes:          attributes,
	}
}

func (r *IfBridgeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.hostData, _ = req.ProviderData.(*linuxhost_client.HostData)
}

func convertIfBridgeResourceModel(ctx context.Context, getter Getter) (*models.IfBridgeResourceModel, *linuxhost_client.IfBridge, *diag.Diagnostics) {
	tflog.Debug(ctx, "converting IfBridgeResourceModel")
	resource, internalBase, diags := ExtractIfResourceModel[*models.IfBridgeResourceModel](ctx, getter)
	if diags.HasError() {
		tflog.Debug(ctx, "Error converting IfBridgeResourceModel")
		return nil, nil, diags
	}
	tflog.Debug(ctx, "did base conversion successfully")

	internal := &linuxhost_client.IfBridge{
		IfCommon: *internalBase,
	}
	return resource, internal, diags
}

func convertBridgeIf(m *models.IfCommonResourceModel, a *linuxhost_client.AdapterInfo, all *linuxhost_client.AdapterInfoSlice) *models.IfBridgeResourceModel {
	rm := &models.IfBridgeResourceModel{
		IfCommonResourceModel: *m,
	}
	// Get to see if bridge has members
	members := all.SelectWithDesignatedBridge(&a.BridgeInfo.BridgeId)
	if len(members) == 0 {
		rm.State = types.StringValue("up")
	}
	return rm
}

func (r *IfBridgeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resourceModel, internal, diags := convertIfBridgeResourceModel(ctx, &req.Plan)
	resp.Diagnostics.Append(*diags...)

	if diags.HasError() {
		tflog.Error(ctx, "Exiting due to error")
		if resp.Diagnostics.HasError() {
			tflog.Error(ctx, "diags has error")
		} else {
			tflog.Info(ctx, "diags has no error")
		}
		return
	}

	_, err := linuxhost_client.CreateIfBridge(r.hostData.Client, internal)

	if err != nil {
		resp.Diagnostics.AddError("Failed creating Bridge", err.Error())
		return
	}

	resp.Diagnostics.Append(IfToState(
		r.hostData, resourceModel, ctx, &resp.State,
		convertBridgeIf)...)
}

func (r *IfBridgeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resourceModel, _, diags := convertIfBridgeResourceModel(ctx, &req.State)

	resp.Diagnostics.Append(*diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(IfToState(
		r.hostData, resourceModel, ctx, &resp.State,
		convertBridgeIf)...)
}

func (r *IfBridgeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	desiredM, desired, diagsA := convertIfBridgeResourceModel(ctx, &req.Plan)
	resp.Diagnostics.Append(*diagsA...)
	_, state, diagsB := convertIfBridgeResourceModel(ctx, &req.State)

	resp.Diagnostics.Append(*diagsB...)
	if resp.Diagnostics.HasError() {
		return
	}
	UpdateIf(r.hostData, desired, state, ctx, &resp.State)

	resp.Diagnostics.Append(resp.State.Set(ctx, desiredM)...)

}
func (r *IfBridgeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.IfBridgeResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	resourceModel, _, diags := convertIfBridgeResourceModel(ctx, &req.State)
	resp.Diagnostics.Append(*diags...)
	linuxhost_client.DeleteInterface(r.hostData.Client, resourceModel.Name.ValueString())

}
