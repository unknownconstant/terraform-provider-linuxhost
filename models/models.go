package linuxhost_models

import "github.com/hashicorp/terraform-plugin-framework/types"

type NetworkInterfaceResourceModel struct {
	Name            types.String `tfsdk:"name"`
	Mac             types.String `tfsdk:"mac"`
	IP4s            types.Set    `tfsdk:"ipv4"`
	Up              types.Bool   `tfsdk:"up"`
	Type            types.String `tfsdk:"type"`
	ParentInterface types.String `tfsdk:"parent_interface"`
	VLAN_id         types.Number `tfsdk:"vlan_id"`
	DHCP            types.String `tfsdk:"dhcp"`
}

func (r *NetworkInterfaceResourceModel) UpString() string {
	str := ""
	if r.Up.ValueBool() {
		str = "up"
	} else {
		str = "down"
	}
	return str
}

type IfCommonResourceModel struct {
	Name   types.String                 `tfsdk:"name"`
	Mac    types.String                 `tfsdk:"mac"`
	IP4s   types.Set                    `tfsdk:"ipv4"`
	State  types.String                 `tfsdk:"state"`
	Bridge *IfBridgeMemberResourceModel `tfsdk:"bridge"`
}
type IfBridgeMemberResourceModel struct {
	Name types.String `tfsdk:"name"`
}

type IsIfResourceModel interface {
	GetCommon() *IfCommonResourceModel
}

// Bridge
type IfBridgeResourceModel struct {
	IfCommonResourceModel
}

var _ IsIfResourceModel = &IfBridgeResourceModel{}

func (m *IfBridgeResourceModel) GetCommon() *IfCommonResourceModel {
	return &m.IfCommonResourceModel
}

// VXLAN
type IfVxlanResourceModel struct {
	IfCommonResourceModel
	Vni  types.Int64 `tfsdk:"vni"`
	Port types.Int32 `tfsdk:"port"`
}

var _ IsIfResourceModel = &IfVxlanResourceModel{}

func (m *IfVxlanResourceModel) GetCommon() *IfCommonResourceModel {
	return &m.IfCommonResourceModel
}

type NetowrkInterfaceIPAssignmentModel struct {
	InterfaceName types.String `tfsdk:"interface_name"`
	IPv4          types.String `tfsdk:"ipv4"`
}

type UserModel struct {
	Username      types.String `tfsdk:"username"`
	UID           types.Number `tfsdk:"uid"`
	GID           types.Number `tfsdk:"gid"`
	PrimaryGroup  types.String `tfsdk:"primary_group"`
	Groups        types.Set    `tfsdk:"groups"`
	HomeDirectory types.String `tfsdk:"home_directory"`
	Shell         types.String `tfsdk:"shell"`

	// Read-only
	Hostname types.String `tfsdk:"hostname"`
}

type GroupModel struct {
	GID     types.Number `tfsdk:"gid"`
	Name    types.String `tfsdk:"name"`
	Members types.Set    `tfsdk:"members"`
}

type CaCertificateModel struct {
	Name              types.String `tfsdk:"name"`
	Source            types.String `tfsdk:"source"`
	FingerprintSha256 types.String `tfsdk:"fingerprint_sha256"`
}
