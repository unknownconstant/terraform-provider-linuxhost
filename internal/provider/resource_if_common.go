package provider

import (
	"context"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func commonInterfaceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "The interface identifier, .e.g. 'eth0'",
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
			ElementType:   types.StringType,
			Computed:      true,
			PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
		},
		"state": schema.StringAttribute{
			Description: "Interface state. Valid options: 'up', 'down'.",
			Optional:    true,
			Computed:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("up", "down"),
			},
		},
		"bridge": schema.SingleNestedAttribute{
			Optional: true,
			MarkdownDescription: "If specified, the bridge this interface is a member of.",
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "The name of the bridge, e.g. 'br0'",
					Required: true,
				},
			},
		},
	}
}

// func GenericCreate[Model InterfaceConfig](
//
//	ctx context.Context,
//	req resource.CreateRequest,
//	resp *resource.CreateResponse,
//	parse func(ctx context.Context, req resource.CreateRequest) (Model, diag.Diagnostics),
//	createFunc func(ctx context.Context, config Model) error,
//	setState func(ctx context.Context, config Model, resp *resource.CreateResponse),
//
// )
type Getter interface {
	Get(context.Context, any) diag.Diagnostics
}
type Setter interface {
	Set(context.Context, any) diag.Diagnostics
	RemoveResource(context.Context)
}

func GetModel[T any](ctx context.Context, model *T, g Getter) diag.Diagnostics {
	return g.Get(ctx, model)
}
func SetModel[T any](ctx context.Context, model *T, s Setter) diag.Diagnostics {
	return s.Set(ctx, model)
}

func BuildIfInternalCommon(resource models.IsIfResourceModel, diags *diag.Diagnostics) *linuxhost_client.IfCommon {
	common := resource.GetCommon()
	internal := &linuxhost_client.IfCommon{}

	if common.Name.IsUnknown() || common.Name.IsNull() {
		diags.AddError("Missing name", "The 'name' field is required.")
	} else {
		internal.Name = common.Name.ValueString()
	}

	if common.State.IsUnknown() || common.State.IsNull() {
		internal.State = ""
	} else {
		internal.State = common.State.ValueString()
	}

	if common.Bridge != nil {
		internal.BridgeMember = &linuxhost_client.IfBridgeMember{
			Name: common.Bridge.Name.ValueString(),
		}
	}
	return internal
}

func ExtractIfResourceModel[T models.IsIfResourceModel](ctx context.Context, getter Getter) (T, *linuxhost_client.IfCommon, *diag.Diagnostics) {
	var full T
	var diags diag.Diagnostics

	diags = GetModel(ctx, &full, getter)
	if diags.HasError() {
		tflog.Error(ctx, "Error getting plan")
		return full, &linuxhost_client.IfCommon{}, &diags
	}

	tflog.Debug(ctx, "Got plan")

	internal := BuildIfInternalCommon(full, &diags)

	return full, internal, &diags
}

func BuildIfResourceModelFromInternal(
	// adapters, _ = linuxhost_client.ReadAdapters(hostData)
	ctx context.Context, adapters *linuxhost_client.AdapterInfoSlice, interfaceDescription *linuxhost_client.AdapterInfo) (*models.IfCommonResourceModel, diag.Diagnostics) {

	diags := diag.Diagnostics{}

	if interfaceDescription == nil {
		return nil, diags
	}
	tflog.Debug(ctx, "Got adapter "+interfaceDescription.Name)

	// Set State
	interfaceState := "down"
	if interfaceDescription.Up == true {
		interfaceState = "up"
	}

	// Set IPv4
	ipv4Strings := []string{}
	for _, ipPair := range interfaceDescription.IPv4 {
		ipv4Strings = append(ipv4Strings, ipPair.IP+"/"+ipPair.Subnet)
	}
	ips, _ := types.SetValueFrom(ctx, types.StringType, ipv4Strings)
	interfaceIPv4s, _ := types.SetValueFrom(ctx, types.StringType, ips)

	commonResourceModel := &models.IfCommonResourceModel{
		Name:  types.StringValue(interfaceDescription.Name),
		Mac:   types.StringValue(interfaceDescription.MAC),
		State: types.StringValue(interfaceState),
		IP4s:  interfaceIPv4s,
	}
	if interfaceDescription.DesignatedBridge != nil {
		// Find the bridge
		bridge := adapters.GetIsBridgeId(*interfaceDescription.DesignatedBridge)
		if bridge == nil {
			diags.AddError(("Could not find bridge for " + interfaceDescription.Name), "Could not find bridge for "+interfaceDescription.Name)
			return nil, diags

		}
		commonResourceModel.Bridge = &models.IfBridgeMemberResourceModel{
			Name: types.StringValue(bridge.Name),
		}
	}
	return commonResourceModel, diags
}

func IfToState[RM models.IsIfResourceModel](
	hostData *linuxhost_client.HostData,
	resourceModel RM,
	ctx context.Context,
	setter Setter,
	provideFinalState func(m *models.IfCommonResourceModel, a *linuxhost_client.AdapterInfo, all *linuxhost_client.AdapterInfoSlice) RM,
) diag.Diagnostics {
	adapters, _ := linuxhost_client.ReadAdapters(hostData)
	interfaceDescription := adapters.GetByName(resourceModel.GetCommon().Name.ValueString())
	commonResourceModel, diags := BuildIfResourceModelFromInternal(ctx, &adapters, interfaceDescription)

	if diags.HasError() {
		return diags
	}

	if commonResourceModel == nil {
		setter.RemoveResource(ctx)
		return diags
	}

	finalModel := provideFinalState(commonResourceModel, interfaceDescription, &adapters)
	return setter.Set(ctx, finalModel)

}

func UpdateIf[M linuxhost_client.IsIf](
	hostData *linuxhost_client.HostData,
	modelDesired M,
	modelState M,
	ctx context.Context,
	setter Setter,
) {
	desired := modelDesired.GetCommon()
	state := modelState.GetCommon()
	if state.State != desired.State {
		linuxhost_client.IfSetState(hostData.Client, modelDesired)
	}
	if &state.BridgeMember != &desired.BridgeMember {
		tflog.Info(ctx, "Bridge member changed")
		linuxhost_client.IfSetBridgeMaster(hostData.Client, modelDesired)
	}
}
