package provider

import (
	"context"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/davecgh/go-spew/spew"
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

func ConvertIfResourceModel[T models.IsIfResourceModel](ctx context.Context, getter Getter) (T, *linuxhost_client.IfCommon, *diag.Diagnostics) {
	var full T
	var diags diag.Diagnostics

	// diags = req.Plan.Get(ctx, &full)
	diags = GetModel(ctx, &full, getter)
	if diags.HasError() {
		tflog.Error(ctx, "Error getting plan")
		return full, &linuxhost_client.IfCommon{}, &diags
	}

	tflog.Debug(ctx, "Got plan")

	tflog.Debug(ctx, spew.Sdump(full))
	c := full.GetCommon()

	internal := &linuxhost_client.IfCommon{}

	if c.Name.IsUnknown() || c.Name.IsNull() {
		diags.AddError("Missing name", "The 'name' field is required.")
	} else {
		internal.Name = c.Name.ValueString()
	}

	if c.State.IsUnknown() || c.State.IsNull() {
		internal.State = ""
	} else {
		internal.State = c.State.ValueString()
	}

	// if c.Mac.IsUnknown() || c.Mac.IsNull() {
	// 	internal.Mac = ""
	// } else {
	// 	internal.Mac = c.Mac.ValueString()
	// }
	// if c.IP4s.IsUnknown() || c.IP4s.IsNull() {
	// 	internal.IPv4 = []models.IPWithSubnet{}
	// 	} else {
	// 	internal.IPv4 = []models.IPWithSubnet{}
	// }

	return full, internal, &diags
}

// func CommonIfReadBack[RM models.IsIfResourceModel, R IsLinuxhostCommonResource](r R)

func ReadSingleIf[RM models.IsIfResourceModel, R IsLinuxhostIFResource](
	resource R,
	resourceModel RM,
	ctx context.Context,
	setter Setter,
	provideFinalState func(m *models.IfCommonResourceModel, a *linuxhost_client.AdapterInfo) RM) diag.Diagnostics {
	adapters, _ := linuxhost_client.ReadAdapters(resource.GetHostData())

	for _, a := range adapters {
		tflog.Debug(ctx, "Got adapter "+a.Name)
		if a.Name != resourceModel.GetCommon().Name.ValueString() {
			continue
		}

		// Set State
		state := "down"
		if a.Up == true {
			state = "up"
		}

		// Set IPv4
		ipv4Strings := []string{}
		for _, ipPair := range a.IPv4 {
			ipv4Strings = append(ipv4Strings, ipPair.IP+"/"+ipPair.Subnet)
		}
		ips, _ := types.SetValueFrom(ctx, types.StringType, ipv4Strings)
		i, _ := types.SetValueFrom(ctx, types.StringType, ips)

		m := &models.IfCommonResourceModel{
			Name:  types.StringValue(a.Name),
			Mac:   types.StringValue(a.MAC),
			State: types.StringValue(state),
			IP4s:  i,
		}
		finalModel := provideFinalState(m, &a)
		return setter.Set(ctx, finalModel)
		// return &diag
		// resp.Diagnostics.Append(resp.State.Set(ctx, finalModel)...)
	}
	setter.RemoveResource(ctx)
	var d diag.Diagnostics
	return d
}

func UpdateIf[M linuxhost_client.IsIf, R IsLinuxhostIFResource](
	resource R,
	modelDesired M,
	modelState M,
	ctx context.Context,
	setter Setter,
) {
	desired := modelDesired.GetCommon()
	state := modelState.GetCommon()
	if state.State != desired.State {
		linuxhost_client.IfSetState(resource.GetHostData().Client, modelDesired)
	}
}
