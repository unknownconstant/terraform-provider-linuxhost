// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
// var _ resource.Resource = &NetworkInterfaceResource{}
// var _ resource.ResourceWithImportState = &NetworkInterfaceResource{}
var _ resource.ResourceWithConfigure = &NetworkInterfaceResource{}
var _ resource.ResourceWithUpgradeState = &NetworkInterfaceResource{}

func NewNetworkInterfaceResource() resource.Resource {
	return &NetworkInterfaceResource{}
}

// NetworkInterfaceResource defines the resource implementation.
type NetworkInterfaceResource struct {
	// client *http.Client
	hostData *linuxhost_client.HostData
}

// NetworkInterfaceResourceModel describes the resource data model.

func (r *NetworkInterfaceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_interface"
}

func (r *NetworkInterfaceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			// "configurable_attribute": schema.StringAttribute{
			// 	MarkdownDescription: "Example configurable attribute",
			// 	Optional:            true,
			// },
			// "defaulted": schema.StringAttribute{
			// 	MarkdownDescription: "Example configurable attribute with default value",
			// 	Optional:            true,
			// 	Computed:            true,
			// 	Default:             stringdefault.StaticString("example value when not configured"),
			// },
			"name": schema.StringAttribute{
				// Computed:            true,
				Optional:            true,
				MarkdownDescription: "Example identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mac": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The assigned interface mac address",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ipv4": schema.SetAttribute{
				ElementType: types.StringType, // A plain set of strings
				// Optional:    true,
				Computed:      true,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				// Default:  setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				// types.SetValueMust(types.StringType, []string{}),
			},
			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("dummy"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vlan_id": schema.NumberAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Number{
					numberplanmodifier.RequiresReplace(),
				},
			},
			"parent_interface": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"up": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				// PlanModifiers: []planmodifier.Bool{
				// 	boolplanmodifier,
				// },
			},
			"dhcp": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("dhclient"),
				},
			},
			// "ipv4": schema.SetAttribute{
			// 	Optional:    true,
			// 	ElementType: types.StringType,
			// 	// ElementType: types.SetType{
			// 	// 	ElemType: types.StringType,
			// 	// },
			// },
		},
		Version: 2,
	}
}

func (r *NetworkInterfaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		tflog.Warn(ctx, "Provider data is missing")
		return
	}
	tflog.Info(ctx, "Provider data is present")

	hostData, ok := req.ProviderData.(*linuxhost_client.HostData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.hostData = hostData

}
func (r *NetworkInterfaceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config models.NetworkInterfaceResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Perform cross-attribute validation
	if config.Type.ValueString() == "vlan" && config.VLAN_id.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("vlan_id"),
			"Missing VLAN ID",
			"When 'type' is set to 'vlan', the 'vlan_id' attribute must be provided.",
		)
	}
	// Perform cross-attribute validation
	if config.Type.ValueString() == "vlan" && config.ParentInterface.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("parent_interface"),
			"Missing parent_interface",
			"When 'type' is set to 'vlan', the 'parent_interface' attribute must be provided.",
		)
	}

	if config.Type.ValueString() == "dummy" && !config.VLAN_id.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("vlan_id"),
			"Unexpected VLAN ID",
			"When 'type' is set to 'dummy', the 'vlan_id' attribute must not be provided.",
		)
	}

	if config.Type.ValueString() == "dummy" && !config.ParentInterface.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("vlan_id"),
			"Unexpected VLAN ID",
			"When 'type' is set to 'dummy', the 'parent_interface' attribute must not be provided.",
		)
	}

}

func (r *NetworkInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.NetworkInterfaceResourceModel

	// fmt.Println("Going to create...")
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	fmt.Println("setting float in interface")
	var i int
	if data.VLAN_id.IsNull() {
		// f = nil
		i = -1
	} else {
		f, _ := data.VLAN_id.ValueBigFloat().Float64()
		i = int(f)
	}
	// f, _ = data.VLAN_id.ValueBigFloat().Float64()
	NI := &linuxhost_client.NetworkInterface{
		Id:              data.Name.String(),
		Type:            data.Type.ValueString(),
		ParentInterface: data.ParentInterface.ValueString(),
		VLAN_id:         i,
	}

	if resp.Diagnostics.HasError() {
		return
	}

	linuxhost_client.CreateInterface(r.hostData.Client, NI)

	if err := linuxhost_client.InterfaceUpDown(r.hostData.Client, NI.Id, data.UpString()); err != nil {
		resp.Diagnostics.AddError("Failed to bring up interface after creation", err.Error())
	}

	if !data.DHCP.IsNull() {
		handleEnableDHCP(r.hostData.Client, data.Name.ValueString(), data.DHCP.ValueString(), resp.Diagnostics)
	}
	// adapters := r.read(ctx)
	// Read Terraform prior state data into the model

	adapters, _ := linuxhost_client.ReadAdapters(r.hostData)
	for _, s := range adapters {
		tflog.Info(ctx, "Adding:"+s.Name+", mac: "+s.MAC)
		if s.Name != data.Name.ValueString() {
			continue
		}
		ipv4 := s.IPv4
		ip := []string{}
		// if ipv4 != nil {
		for _, ipPair := range ipv4 {
			ip = append(ip, ipPair.IP+"/"+ipPair.Subnet)
		}
		// }
		ips, _ := types.SetValueFrom(ctx, types.StringType, ip)

		i, _ := types.SetValueFrom(ctx, types.StringType, ips)
		// v := &big.Float{}
		// // x := &basetypes.NumberValue{}
		// if s.Vlan == nil {
		// 	v = nil
		// } else {
		// 	fmt.Println("setting float in resource")
		// 	v = new(big.Float).SetFloat64(float64(*s.Vlan))
		// }
		// fmt.Println("ParentInterface value: " + *s.ParentInterface + ";")
		N := &models.NetworkInterfaceResourceModel{
			Name:            types.StringValue(s.Name),
			Mac:             types.StringValue(s.MAC),
			IP4s:            i,
			Up:              types.BoolValue(s.Up),
			Type:            types.StringValue(s.Type),
			ParentInterface: stringOrNull(s.ParentInterface),
			DHCP:            stringOrNull(s.DHCP),
			VLAN_id:         numberOrNull(s.Vlan),
		}
		fmt.Println(N.Name.String())
		resp.Diagnostics.Append(resp.State.Set(ctx, N)...)
	}
}

func (r *NetworkInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Warn(ctx, "read a data source:")
	if r.hostData == nil {
		resp.Diagnostics.AddError("Missing client", "")
		return
	}
	var data models.NetworkInterfaceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	adapters, _ := linuxhost_client.ReadAdapters(r.hostData)
	// adapters := r.read(ctx)
	for _, s := range adapters {
		fmt.Println(s)
		if s.Name != data.Name.ValueString() {
			continue
		}
		tflog.Info(ctx, "Adding:"+s.Name+", mac: "+s.MAC)
		ipv4 := s.IPv4
		ip := []string{}
		// if ipv4 != nil {
		for _, ipPair := range ipv4 {
			ip = append(ip, ipPair.IP+"/"+ipPair.Subnet)
		}
		// }
		ips, _ := types.SetValueFrom(ctx, types.StringType, ip)

		i, _ := types.SetValueFrom(ctx, types.StringType, ips)

		// x := &basetypes.NumberValue{}
		// if s.Vlan == nil {
		// 	v = nil
		// } else {
		// 	fmt.Println("setting float in interface 2")
		// 	v = new(big.Float).SetFloat64(float64(*s.Vlan))
		// }
		// fmt.Println("read ParentInterface value: " + s.ParentInterface + ";")
		N := &models.NetworkInterfaceResourceModel{
			Name:            types.StringValue(s.Name),
			Mac:             types.StringValue(s.MAC),
			IP4s:            i,
			Up:              types.BoolValue(s.Up),
			Type:            types.StringValue(s.Type),
			ParentInterface: stringOrNull(s.ParentInterface),
			DHCP:            stringOrNull(s.DHCP),
			VLAN_id:         numberOrNull(s.Vlan),
		}
		fmt.Println(N.Name.String())
		resp.Diagnostics.Append(resp.State.Set(ctx, N)...)
		return
	}
	resp.State.RemoveResource(ctx)

	if resp.Diagnostics.HasError() {
		return
	}
}

func handleEnableDHCP(Client *linuxhost_client.SSHClientContext, name string, dhcp string, diag diag.Diagnostics) {
	err := linuxhost_client.SetDhcp(Client, name, dhcp, true)
	if err != nil {
		diag.AddError("Failed to enable DHCP on interface", err.Error())
	}

}

func (r *NetworkInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var desired models.NetworkInterfaceResourceModel
	var state models.NetworkInterfaceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &desired)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}
	if !desired.Up.Equal(state.Up) {
		err := linuxhost_client.InterfaceUpDown(r.hostData.Client, desired.Name.ValueString(), desired.UpString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to delete IP from interface", err.Error())
		} else {
			fmt.Println("did ok!")
		}
		fmt.Println(desired)
	}

	if !desired.DHCP.Equal(state.DHCP) {
		if desired.DHCP.IsNull() {
			err := linuxhost_client.SetDhcp(r.hostData.Client, desired.Name.ValueString(), state.DHCP.ValueString(), false)
			if err != nil {
				resp.Diagnostics.AddError("Failed to disable DHCP on interface", err.Error())
			}
		} else {
			handleEnableDHCP(r.hostData.Client, desired.Name.ValueString(), desired.DHCP.ValueString(), resp.Diagnostics)
		}
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &desired)...)
}

func (r *NetworkInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.NetworkInterfaceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if !data.DHCP.IsNull() {
		linuxhost_client.SetDhcp(r.hostData.Client, data.Name.ValueString(), data.DHCP.ValueString(), false)
	}
	linuxhost_client.DeleteInterface(r.hostData.Client, data.Name.String())

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *NetworkInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var V0 = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id": tftypes.String,
	},
}

var V1 = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id":  tftypes.String,
		"mac": tftypes.String,
	},
}
var V2 = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id":   tftypes.String,
		"mac":  tftypes.String,
		"ipv4": tftypes.List{ElementType: tftypes.String},
	},
}
var V3 = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"name": tftypes.String,
		"mac":  tftypes.String,
		"ipv4": tftypes.List{ElementType: tftypes.String},
	},
}
var V4 = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"name": tftypes.String,
		"mac":  tftypes.String,
		"ipv4": tftypes.List{ElementType: tftypes.String},
		"type": tftypes.String,
	},
}

func (r *NetworkInterfaceResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade implementation from 0 (prior state version) to 2 (Schema.Version)
		2: {
			// Optionally, the PriorSchema field can be defined.
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				rawStateValue, err := req.RawState.Unmarshal(V0)
				fmt.Printf("DOING STATE CONVERSION")
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to Unmarshal Prior State",
						err.Error(),
					)
					return
				}

				var rawState map[string]tftypes.Value

				if err := rawStateValue.As(&rawState); err != nil {
					resp.Diagnostics.AddError(
						"Unable to Convert Prior State",
						err.Error(),
					)
					return
				}

				dynamicValue, err := tfprotov6.NewDynamicValue(
					V1,
					tftypes.NewValue(V1, map[string]tftypes.Value{
						"id":  rawState["id"],
						"mac": tftypes.NewValue(tftypes.String, ""),
					}),
				)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to sss",
						err.Error(),
					)
					return
				}
				resp.DynamicValue = &dynamicValue
			},
		},
		3: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				rawStateValue, err := req.RawState.Unmarshal(V2)
				fmt.Printf("DOING STATE CONVERSION")
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to Unmarshal Prior State",
						err.Error(),
					)
					return
				}

				var rawState map[string]tftypes.Value

				if err := rawStateValue.As(&rawState); err != nil {
					resp.Diagnostics.AddError(
						"Unable to Convert Prior State",
						err.Error(),
					)
					return
				}

				dynamicValue, err := tfprotov6.NewDynamicValue(
					V2,
					tftypes.NewValue(V1, map[string]tftypes.Value{
						"id":   rawState["id"],
						"mac":  tftypes.NewValue(tftypes.String, ""),
						"ipv4": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, 2),
						"type": tftypes.NewValue(tftypes.String, "dummy"),
					}),
				)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to sss",
						err.Error(),
					)
					return
				}
				resp.DynamicValue = &dynamicValue
			},
		},

		4: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				rawStateValue, err := req.RawState.Unmarshal(V2)
				fmt.Printf("DOING STATE CONVERSION")
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to Unmarshal Prior State",
						err.Error(),
					)
					return
				}

				var rawState map[string]tftypes.Value

				if err := rawStateValue.As(&rawState); err != nil {
					resp.Diagnostics.AddError(
						"Unable to Convert Prior State",
						err.Error(),
					)
					return
				}

				dynamicValue, err := tfprotov6.NewDynamicValue(
					V2,
					tftypes.NewValue(V4, map[string]tftypes.Value{
						"id":   rawState["id"],
						"mac":  tftypes.NewValue(tftypes.String, ""),
						"ipv4": tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, 2),
						"type": tftypes.NewValue(tftypes.String, "dummy"),
					}),
				)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to sss",
						err.Error(),
					)
					return
				}
				resp.DynamicValue = &dynamicValue

			},
		},
		// // State upgrade implementation from 1 (prior state version) to 2 (Schema.Version)
		// 1: {
		// 		// Optionally, the PriorSchema field can be defined.
		// 		StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) { /* ... */ },
		// },
	}
}
