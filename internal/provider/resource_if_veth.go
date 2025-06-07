package provider

import (
	"context"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &IfVethResource{}
var _ IsLinuxhostIFResource = &IfVethResource{}

type IfVethResource struct {
	LinuxhostCommonResource
}

func NewIfVethResource() resource.Resource {
	return &IfVethResource{}
}

var _ IsLinuxhostIFResource = &IfVethResource{}

func (r *IfVethResource) GetHostData() *linuxhost_client.HostData {
	return r.LinuxhostCommonResource.hostData
}

func (r *IfVethResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_if_veth"
}

func (r *IfVethResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	localAttributes := commonInterfaceSchema()
	peerAttributes := commonInterfaceSchema()
	attributes := map[string]schema.Attribute{
		"local": schema.SingleNestedAttribute{
			Required:   true,
			Attributes: localAttributes,
		},
		"peer": schema.SingleNestedAttribute{
			Required:   true,
			Attributes: peerAttributes,
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Veth interface pair",
		Version:             1,
		Attributes:          attributes,
	}
}

func (r *IfVethResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.hostData, _ = req.ProviderData.(*linuxhost_client.HostData)
}

func extractIfVethResourceModel(ctx context.Context, getter Getter) (*models.IfVethPairResourceModel, *linuxhost_client.IfVethPair, *diag.Diagnostics) {
	var resourceModel *models.IfVethPairResourceModel
	tflog.Debug(ctx, "going to get vethpair model from plan")
	diags := getter.Get(ctx, &resourceModel)
	tflog.Debug(ctx, "got vethpair model from plan")

	internalLocal := BuildIfInternalCommon(resourceModel.Local, &diags)
	tflog.Debug(ctx, "got veth local internal model from resource model")
	internalPeer := BuildIfInternalCommon(resourceModel.Peer, &diags)
	tflog.Debug(ctx, "got veth peer internal model from resource model")

	tflog.Debug(ctx, "converting IfVethResourceModel")
	// resource, internalBase, diags := ExtractIfResourceModel[*models.IfVethResourceModel](ctx, getter)
	if diags.HasError() {
		tflog.Debug(ctx, "Error converting IfVethResourceModel")
		return nil, nil, &diags
	}
	tflog.Debug(ctx, "did base conversion successfully")

	internal := &linuxhost_client.IfVethPair{
		Local: linuxhost_client.IfVethPeer{
			IfCommon: *internalLocal,
		},
		Peer: linuxhost_client.IfVethPeer{
			IfCommon: *internalPeer,
		},
		// IfCommon: *internalBase,
	}
	return resourceModel, internal, &diags
}

func IfVethToState(
	hostData *linuxhost_client.HostData,
	resourceModel *models.IfVethPairResourceModel,
	ctx context.Context,
	setter Setter,
) diag.Diagnostics {
	adapters, _ := linuxhost_client.ReadAdapters(hostData)
	interfaceDescriptionLocal := adapters.GetByName(resourceModel.Local.Name.ValueString())
	if interfaceDescriptionLocal == nil {
		tflog.Debug(ctx, "interfaceDescriptionLocal is nil! Was looking for "+resourceModel.Local.Name.ValueString())
	}
	commonResourceModelLocal, diagsLocal := BuildIfResourceModelFromInternal(ctx, &adapters, interfaceDescriptionLocal)
	if diagsLocal.HasError() {
		tflog.Debug(ctx, "diagsLocal has error")
		return diagsLocal
	}

	interfaceDescriptionPeer := adapters.GetByName(resourceModel.Peer.Name.ValueString())
	if interfaceDescriptionPeer == nil {
		tflog.Debug(ctx, "interfaceDescriptionPeer is nil!")
	}
	commonResourceModelPeer, diagsPeer := BuildIfResourceModelFromInternal(ctx, &adapters, interfaceDescriptionPeer)
	if diagsPeer.HasError() {
		tflog.Debug(ctx, "diagsPeer has error")
		return diagsPeer
	}

	if commonResourceModelLocal == nil || commonResourceModelPeer == nil {
		tflog.Debug(ctx, "commonResourceModelLocal or commonResourceModelPeer is nil")
		if commonResourceModelLocal == nil {
			tflog.Debug(ctx, "commonResourceModelLocal is nil")
		}
		if commonResourceModelPeer == nil {
			tflog.Debug(ctx, "commonResourceModelPeer is nil")
		}
		setter.RemoveResource(ctx)
		diagsLocal.Append(diagsPeer...)
		return diagsLocal
	}

	tflog.Debug(ctx, "setting final model for vethToState")

	finalModel := &models.IfVethPairResourceModel{
		Local: &models.IfVethPeerResourceModel{
			IfCommonResourceModel: *commonResourceModelLocal,
		},
		Peer: &models.IfVethPeerResourceModel{
			IfCommonResourceModel: *commonResourceModelPeer,
		},
	}
	return setter.Set(ctx, finalModel)
}

func (r *IfVethResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resourceModel, internal, diags := extractIfVethResourceModel(ctx, &req.Plan)
	resp.Diagnostics.Append(*diags...)

	tflog.Debug(ctx, "going to create veth pair")
	if diags.HasError() {
		tflog.Error(ctx, "Exiting due to error")
		if resp.Diagnostics.HasError() {
			tflog.Error(ctx, "diags has error")
		} else {
			tflog.Info(ctx, "diags has no error")
		}
		return
	}

	tflog.Debug(ctx, "veth pair about to create")
	_, err := linuxhost_client.CreateIfVeth(r.hostData.Client, internal)
	tflog.Debug(ctx, "veth pair created")
	r.hostData.Interfaces.Clear()
	if r.hostData.Interfaces != nil {
		resp.Diagnostics.AddError("Failed to delete Interfaces from cache", "Failed to delete interfaces from cache")
	}

	if err != nil {
		tflog.Debug(ctx, "Failed creating Veth pair")
		resp.Diagnostics.AddError("Failed creating Veth pair", err.Error())
		return
	}

	resp.Diagnostics.Append(IfVethToState(
		r.hostData, resourceModel, ctx, &resp.State)...)

	// resp.Diagnostics.Append(IfToState(
	// 	r.hostData, resourceModel.Peer, ctx, &resp.State,
	// 	convertVethIf)...)
}
func (r *IfVethResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resourceModel, _, diags := extractIfVethResourceModel(ctx, &req.State)

	resp.Diagnostics.Append(*diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(IfVethToState(
		r.hostData, resourceModel, ctx, &resp.State)...)
}

func (r *IfVethResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	desiredM, desired, diagsA := extractIfVethResourceModel(ctx, &req.Plan)
	resp.Diagnostics.Append(*diagsA...)
	_, state, diagsB := extractIfVethResourceModel(ctx, &req.State)

	resp.Diagnostics.Append(*diagsB...)
	if resp.Diagnostics.HasError() {
		return
	}
	UpdateIf(r.hostData, &desired.Local, &state.Local, ctx, &resp.State)
	UpdateIf(r.hostData, &desired.Peer, &state.Peer, ctx, &resp.State)

	resp.Diagnostics.Append(resp.State.Set(ctx, desiredM)...)

}
func (r *IfVethResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.IfVethPairResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	resourceModel, _, diags := extractIfVethResourceModel(ctx, &req.State)
	resp.Diagnostics.Append(*diags...)
	linuxhost_client.DeleteInterface(r.hostData.Client, resourceModel.Local.Name.ValueString())

}
