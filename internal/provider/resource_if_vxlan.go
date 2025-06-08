package provider

import (
	"context"
	"fmt"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &IfVxlanResource{}

// var _ resource.ResourceWithUpgradeState = &IfVxlanResource{}

func NewIfVxlanResource() resource.Resource {
	return &IfVxlanResource{}
}

type IfVxlanResource struct {
	LinuxhostCommonResource
}

var _ IsLinuxhostIFResource = &IfVxlanResource{}

func (r *IfVxlanResource) GetHostData() *linuxhost_client.HostData {
	return r.hostData
}

func (r *IfVxlanResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_if_vxlan"
}

func (r *IfVxlanResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := commonInterfaceSchema()
	attributes["vni"] = schema.Int64Attribute{
		Required: true,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.RequiresReplace(),
		},
	}
	attributes["port"] = schema.Int32Attribute{
		Optional: true,
		Computed: true,
		Default:  int32default.StaticInt32(4789),
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A vxlan interface",
		Version:             1,
		Attributes:          attributes,
	}
}

func (r *IfVxlanResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.hostData, _ = req.ProviderData.(*linuxhost_client.HostData)
}

func convertIfVxlanResourceModel(ctx context.Context, getter Getter) (*models.IfVxlanResourceModel, *linuxhost_client.IfVxlan, *diag.Diagnostics) {
	// var resourceModel models.IfVxlanResourceModel
	// resp.Diagnostics.Append(req.Plan.Get(ctx, &resourceModel)...)
	tflog.Debug(ctx, "converting IfVxlanResourceModel")

	resource, internalBase, diags := ExtractIfResourceModel[*models.IfVxlanResourceModel](ctx, getter)
	if diags.HasError() {
		tflog.Debug(ctx, "Error converting IfVxlanResourceModel")
		return nil, nil, diags
	}
	tflog.Debug(ctx, "did base conversion successfully")
	tflog.Debug(ctx, spew.Sdump(resource))
	if resource.Vni.IsUnknown() {
		diags.AddError("Missing Vni", "Vni is missing")
	}
	if resource.Vni.IsNull() {
		tflog.Error(ctx, "Vni is null")
	}
	tflog.Debug(ctx, fmt.Sprintf("will convert vni %d", resource.Vni.ValueInt64()))

	internal := &linuxhost_client.IfVxlan{
		IfCommon: *internalBase,
		Vni:      uint32(resource.Vni.ValueInt64()),
		Port:     uint32(resource.Port.ValueInt32()),
	}

	return resource, internal, diags
}
func convertVxlanIf(m *models.IfCommonResourceModel, a *linuxhost_client.AdapterInfo, all *linuxhost_client.AdapterInfoSlice) *models.IfVxlanResourceModel {
	rm := &models.IfVxlanResourceModel{
		IfCommonResourceModel: *m,
	}
	rm.Vni = int64OrNull(a.Vni)
	rm.Port = int32OrNull(a.Port)
	return rm
}

func (r *IfVxlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resourceModel, internal, diags := convertIfVxlanResourceModel(ctx, &req.Plan)
	resp.Diagnostics.Append(*diags...)

	// var data models.IfVxlanModel;
	if diags.HasError() {
		tflog.Error(ctx, "Exiting due to error")
		if resp.Diagnostics.HasError() {
			tflog.Error(ctx, "diags has error")
		} else {
			tflog.Info(ctx, "diags has no error")
		}
		return
	}

	_, err := linuxhost_client.CreateIfVXLAN(r.hostData.Client, internal)

	if err != nil {
		resp.Diagnostics.AddError("Failed creating vxlan", err.Error())
		return
	}
	r.hostData.Interfaces.Clear()
	if r.hostData.Interfaces != nil {
		resp.Diagnostics.AddError("Failed to delete Interfaces from cache", "Failed to delete interfaces from cache")
	}

	diags.AddWarning("Resource vxlan created", resourceModel.Name.ValueString())

	resp.Diagnostics.Append(IfToState(
		r.hostData, resourceModel, ctx, &resp.State,
		convertVxlanIf)...)

	// CommonIf(&data, resp.Diagnostics)

	// GenericCreate[models.IfCommonModel](
	// 	ctx, req, resp,
	// )
	// var data models.IfVxlanModel
	// resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	// if resp.Diagnostics.HasError() {
	// 	return
	// }
	// linuxhost_client.SetVxlan(r.hostData.Client, &data, nil)
	// r.makeStateRefresher(ctx, &resp.State, &resp.Diagnostics).InState(data, "present")
}
func (r *IfVxlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.IfVxlanResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	desiredM, desired, diagsA := convertIfVxlanResourceModel(ctx, &req.Plan)
	resp.Diagnostics.Append(*diagsA...)
	_, state, diagsB := convertIfVxlanResourceModel(ctx, &req.State)

	resp.Diagnostics.Append(*diagsB...)
	if resp.Diagnostics.HasError() {
		return
	}
	UpdateIf(r.hostData, desired, state, ctx, &resp.State)

	resp.Diagnostics.Append(resp.State.Set(ctx, desiredM)...)

	// linuxhost_client.SetVxlan(r.hostData.Client, &data, nil)
	// r.makeStateRefresher(ctx, &resp.State, &resp.Diagnostics).InState(data, "present")
}
func (r *IfVxlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.IfVxlanResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	resourceModel, _, diags := convertIfVxlanResourceModel(ctx, &req.State)
	resp.Diagnostics.Append(*diags...)
	linuxhost_client.DeleteInterface(r.hostData.Client, resourceModel.Name.ValueString())
}

func (r *IfVxlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resourceModel, _, diags := convertIfVxlanResourceModel(ctx, &req.State)

	resp.Diagnostics.Append(*diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(IfToState(
		r.hostData, resourceModel, ctx, &resp.State,
		convertVxlanIf)...)

}

func (r *IfVxlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
