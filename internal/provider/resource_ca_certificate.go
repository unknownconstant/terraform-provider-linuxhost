package provider

import (
	"context"
	"encoding/pem"
	"fmt"
	"terraform-provider-linuxhost/linuxhost_client"
	models "terraform-provider-linuxhost/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithConfigure = &CaCertificateResource{}

func NewCaCertificateResource() resource.Resource {
	return &CaCertificateResource{}
}

type CaCertificateResource struct {
	hostData *linuxhost_client.HostData
}

func (r *CaCertificateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ca_certificate"
}

func (r *CaCertificateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A trusted root certificate on the host",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable name for the certificate, also used as its filename",
			},
			"certificate": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The pem encoded certificate. If supplied, this will be used as the certificate source. Otherwise, it is populated by the value of 'source'.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The certificate source location. For a file, a standard unix path. Or, https://example.com/certificate.pem.",
			},
			"fingerprint_sha256": schema.StringAttribute{
				Computed: true,
			},
			"serial_number": schema.StringAttribute{
				Computed: true,
			},
		},
		Version: 1,
	}
}

func (r *CaCertificateResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("certificate"),
			path.MatchRoot("source"),
		),
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("certificate"),
			path.MatchRoot("source"),
		),
	}
}

func (r *CaCertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.hostData, _ = req.ProviderData.(*linuxhost_client.HostData)
}

const serialNumberSize = 20

func (r *CaCertificateResource) readState(ctx context.Context, data *models.CaCertificateModel, State *tfsdk.State, Diagnostics *diag.Diagnostics, expect string) {
	content, _ := linuxhost_client.CertificateContent(*data)
	expected := linuxhost_client.CertificateInfo(*content)

	certs := linuxhost_client.RefreshRemoteCertificates(r.hostData.Client)

	for _, cert := range certs {
		fingerprint := linuxhost_client.Sha256Fingerprint(cert)
		var serialNumber [serialNumberSize]byte
		serialBytes := cert.SerialNumber.Bytes()
		if len(serialBytes) <= serialNumberSize {
			copy(serialNumber[serialNumberSize-len(serialBytes):], serialBytes)
		}
		if fingerprint != expected.Sha256Fingerprint {
			continue
		}
		pemString := string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		}))
		crt := &models.CaCertificateModel{
			Name:              data.Name,
			Source:            data.Source,
			FingerprintSha256: types.StringValue(linuxhost_client.EncodeBytesString(fingerprint[:])),
			SerialNumber:      types.StringValue(linuxhost_client.EncodeBytesString(serialNumber[:])),
			Certificate:       types.StringValue(pemString),
		}
		if expect == "absent" {
			Diagnostics.AddError("Failed to delete", "The delete operation did not report any errors but the resource remains present in the reported state.")
			return
		}
		Diagnostics.Append(State.Set(ctx, crt)...)
		return
	}
	if expect == "present" {
		Diagnostics.AddError("Didn't find certificate", "")
	} else if expect == "any" {
		State.RemoveResource(ctx)
	} else if expect == "absent" {
		State.RemoveResource(ctx)
	} else {
		Diagnostics.AddError("Invalid expectation", "This is an error with the provider 'linuxhost'")
	}
}

func (r *CaCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.CaCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	DestinationPath := fmt.Sprintf("/usr/local/share/ca-certificates/%s.crt", data.Name)
	Permissions := 0o600
	Uid := 0
	Gid := 0

	commandContext := linuxhost_client.NewSSHCommandContext(r.hostData.Client)
	var result *linuxhost_client.SSHCommandContext
	content, _ := linuxhost_client.CertificateContent(data)

	// if data.Source.IsUnknown() || data.Source.IsNull() {
	params := &linuxhost_client.FileContentParams{
		Content:         *content,
		DestinationPath: DestinationPath,
		Permissions:     &Permissions,
		Uid:             &Uid,
		Gid:             &Gid,
	}
	result = linuxhost_client.SetTextFileContent(&commandContext, params)
	// } else {
	// params := &linuxhost_client.FileTransferParams{
	// 	SourcePath:      data.Source.ValueString(),
	// 	DestinationPath: DestinationPath,
	// 	Permissions:     &Permissions,
	// 	Uid:             &Uid,
	// 	Gid:             &Gid,
	// }
	// result = linuxhost_client.SetTextFile(&commandContext, params)
	// }
	if result.Error != nil {
		resp.Diagnostics.AddError("An error occurred", result.Error.Error())
		return
	}
	linuxhost_client.SetRemoteCaTrust(*result)
	r.readState(ctx, &data, &resp.State, &resp.Diagnostics, "present")
}

func (r *CaCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.hostData == nil {
		resp.Diagnostics.AddError("Missing client", "")
		return
	}
	var data models.CaCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	r.readState(ctx, &data, &resp.State, &resp.Diagnostics, "any")

	// if resp.Diagnostics.HasError() {
	// 	return
	// }
}

func (r *CaCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.CaCertificateModel
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

func (r *CaCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.CaCertificateModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	commandContext := linuxhost_client.NewSSHCommandContext(r.hostData.Client).
		Exec(fmt.Sprintf("sudo rm /usr/local/share/ca-certificates/%s.crt", data.Name))
	commandContext = linuxhost_client.SetRemoteCaTrust(commandContext)
	if commandContext.Error != nil {
		resp.Diagnostics.AddError("Failed to delete CA certificate", commandContext.Error.Error())
	}

	r.readState(ctx, &data, &resp.State, &resp.Diagnostics, "absent")

	// resp.Diagnostics.AddError("Not implemented", "Delete is not implemented.")
}
func (r *CaCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
