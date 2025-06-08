package provider

import (
	"context"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &IfVlanResource{}
var _ IsLinuxhostIFResource = &IfVlanResource{}

func NewIfVlanResource() resource.Resource {
	return &IfVlanResource{}
}

type IfVlanResource struct {
	LinuxhostCommonResource
}

func (r *IfVlanResource) GetHostData() *linuxhost_client.HostData {
	return r.hostData
}

func (r *IfVlanResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_if_vlan"
}

func (r *IfVlanResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := commonInterfaceSchema()
	attributes["parent"] = schema.StringAttribute{
		MarkdownDescription: "The parent interface, e.g. eth0",
		Required:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	attributes["vid"] = schema.Int32Attribute{
		MarkdownDescription: "The VLAN ID",
		Required:            true,
		PlanModifiers: []planmodifier.Int32{
			int32planmodifier.RequiresReplace(),
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A vlan interface",
		Version:             1,
		Attributes:          attributes,
	}
}

func (r *IfVlanResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.hostData, _ = req.ProviderData.(*linuxhost_client.HostData)
}

func convertIfVlanResourceModel(ctx context.Context, getter Getter) (*models.IfVlanResourceModel, *linuxhost_client.IfVlan, *diag.Diagnostics) {
	tflog.Debug(ctx, "converting IfVlanResourceModel")

	resource, internalBase, diags := ExtractIfResourceModel[*models.IfVlanResourceModel](ctx, getter)
	if diags.HasError() {
		tflog.Debug(ctx, "Error converting IfVlanResourceModel")
		return nil, nil, diags
	}
	tflog.Debug(ctx, "did base conversion successfully")
	tflog.Debug(ctx, spew.Sdump(resource))

	internal := &linuxhost_client.IfVlan{
		IfCommon: *internalBase,
		Vid:      uint32(resource.Vid.ValueInt32()),
		Parent:   resource.Parent.ValueString(),
	}

	return resource, internal, diags
}
func convertVlanIf(m *models.IfCommonResourceModel, a *linuxhost_client.AdapterInfo, all *linuxhost_client.AdapterInfoSlice) *models.IfVlanResourceModel {
	rm := &models.IfVlanResourceModel{
		IfCommonResourceModel: *m,
	}
	if(a.VlanInfo == nil) {
		return nil
	}
	if(a.VlanInfo.Parent == "") {
		return nil
	}
	if(a.VlanInfo.Vid == 0) {
		return nil
	}
	rm.Vid = int32OrNull(a.VlanInfo.Vid)
	if(rm.Vid.IsNull()){
		return nil
	}
	rm.Parent = stringOrNull(a.VlanInfo.Parent)
	return rm
}

func (r *IfVlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resourceModel, internal, diags := convertIfVlanResourceModel(ctx, &req.Plan)
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

	_, err := linuxhost_client.CreateIfVlan(r.hostData.Client, internal)

	if err != nil {
		resp.Diagnostics.AddError("Failed creating vlan", err.Error())
		return
	}
	r.hostData.Interfaces.Clear()
	if r.hostData.Interfaces != nil {
		resp.Diagnostics.AddError("Failed to delete Interfaces from cache", "Failed to delete interfaces from cache")
	}

	diags.AddWarning("Resource vlan created", resourceModel.Name.ValueString())

	resp.Diagnostics.Append(IfToState(
		r.hostData, resourceModel, ctx, &resp.State,
		convertVlanIf)...)
}
func (r *IfVlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.IfVlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	desiredM, desired, diagsA := convertIfVlanResourceModel(ctx, &req.Plan)
	resp.Diagnostics.Append(*diagsA...)
	_, state, diagsB := convertIfVlanResourceModel(ctx, &req.State)

	resp.Diagnostics.Append(*diagsB...)
	if resp.Diagnostics.HasError() {
		return
	}
	UpdateIf(r.hostData, desired, state, ctx, &resp.State)

	resp.Diagnostics.Append(resp.State.Set(ctx, desiredM)...)
}
func (r *IfVlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.IfVlanResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	resourceModel, _, diags := convertIfVlanResourceModel(ctx, &req.State)
	resp.Diagnostics.Append(*diags...)
	linuxhost_client.DeleteInterface(r.hostData.Client, resourceModel.Name.ValueString())
}

func (r *IfVlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resourceModel, _, diags := convertIfVlanResourceModel(ctx, &req.State)

	resp.Diagnostics.Append(*diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(IfToState(
		r.hostData, resourceModel, ctx, &resp.State,
		convertVlanIf)...)

}

func (r *IfVlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// func (r *IfVlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
// 	parts := strings.Split(req.ID, "@")
// 	if len(parts) != 2 {
// 		resp.Diagnostics.AddError("Invalid import format", "Expected format: vlan_interface@parent_interface (e.g. eth0.100@eth0)")
// 		return
// 	}
// 	vlanIf := parts[0]
// 	parentIf := parts[1]

// 	resp.State.SetAttribute(ctx, path.Root("name"), vlanIf)
// 	resp.State.SetAttribute(ctx, path.Root("parent"), parentIf)
// }
