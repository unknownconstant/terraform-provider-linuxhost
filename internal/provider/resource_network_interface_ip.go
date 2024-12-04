package provider

import (
	"context"
	"fmt"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.ResourceWithConfigure = &NetworkInterfaceIPResource{}

func NewNetworkInterfaceIPResource() resource.Resource {
	return &NetworkInterfaceIPResource{}
}

type NetworkInterfaceIPResource struct {
	hostData *linuxhost_client.HostData
}

func (r *NetworkInterfaceIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_interface_ip"
}

func (r *NetworkInterfaceIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An assignment of an IP address to an adapter.",
		Attributes: map[string]schema.Attribute{
			"interface_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the interface to assign the IP address to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ipv4": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The IP to assign as 0.0.0.0/0",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Version: 1,
	}
}

func (r *NetworkInterfaceIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.hostData, _ = req.ProviderData.(*linuxhost_client.HostData)
}

func (r *NetworkInterfaceIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.NetowrkInterfaceIPAssignmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := linuxhost_client.AssignIP(r.hostData.Client, data.InterfaceName.String(), data.IPv4.String())
	if err != nil {
		resp.Diagnostics.AddError("Failed when creating IP on "+data.InterfaceName.String(), err.Error())
	}
	adapters, _ := linuxhost_client.RefreshAdapters(r.hostData)

	for _, s := range adapters {
		fmt.Println(s.Name + ":::" + data.InterfaceName.ValueString())
		if s.Name != data.InterfaceName.ValueString() {
			continue
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
		return
	}
	resp.Diagnostics.AddError("Failed to read adapter", "didn't find adapter "+data.InterfaceName.String())
}

func (r *NetworkInterfaceIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.hostData == nil {
		resp.Diagnostics.AddError("Missing client", "")
		return
	}
	var data models.NetowrkInterfaceIPAssignmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	tflog.Warn(ctx, "IP::: read a data source: "+data.InterfaceName.String())

	adapters, _ := linuxhost_client.ReadAdapters(r.hostData)
	for _, s := range adapters {
		if s.Name != data.InterfaceName.ValueString() {
			continue
		}
		tflog.Info(ctx, "IP:("+data.InterfaceName.String()+", "+data.IPv4.String()+"):: FOUND "+s.Name+", mac: "+s.MAC)
		tflog.Info(ctx, "IP:("+data.InterfaceName.String()+", "+data.IPv4.String()+"):: Looking for "+data.IPv4.String())
		ipv4 := ""
		for _, ipPair := range s.IPv4 {
			ip := ipPair.String()
			tflog.Info(ctx, "IP:("+data.InterfaceName.String()+", "+data.IPv4.String()+"):: Checking "+data.IPv4.String()+", "+ip)
			if ip != data.IPv4.ValueString() {
				continue
			}
			ipv4 = ip
			break
		}
		// resp.Diagnostics.AddError("sdfs", s.Name+", "+s.MAC+", "+strings.Join(models.IPList(s.IPv4), ", "))
		if ipv4 == "" {
			// resp.Diagnostics.AddError("("+data.InterfaceName.String()+", "+data.IPv4.String()+") Couldn't find", s.Name)
			resp.State.RemoveResource(ctx)

			return
		} else {
			N := &models.NetowrkInterfaceIPAssignmentModel{
				InterfaceName: types.StringValue(s.Name),
				IPv4:          types.StringValue(ipv4),
			}
			fmt.Println(N.InterfaceName.String())
			resp.Diagnostics.Append(resp.State.Set(ctx, N)...)
			return
		}
	}
	// resp.Diagnostics.AddError("Couldn't find interface "+data.InterfaceName.String()+", "+data.IPv4.String(), "")
	// resp.Diagnostics.Append(resp.State.Set(ctx, &models.NetowrkInterfaceIPAssignmentModel{})...)
	resp.State.RemoveResource(ctx)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *NetworkInterfaceIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.NetowrkInterfaceIPAssignmentModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError("Not implemented", "Update is not implemented.")
	// return

	// Save updated data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkInterfaceIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.NetowrkInterfaceIPAssignmentModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	err := linuxhost_client.DeleteIP(r.hostData.Client, data.InterfaceName.ValueString(), data.IPv4.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete IP from interface", err.Error())
	}

	if resp.Diagnostics.HasError() {
		return
	}
}
func (r *NetworkInterfaceIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
