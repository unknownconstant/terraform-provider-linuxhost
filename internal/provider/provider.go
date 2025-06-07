// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"terraform-provider-linuxhost/linuxhost_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
// var _ provider.Provider = &ScaffoldingProvider{}
// var _ provider.ProviderWithFunctions = &ScaffoldingProvider{}

// linuxHostProviderModel maps provider schema data to a Go type.
//
//	type linuxHostProviderModel struct {
//		Host     types.String `tfsdk:"host"`
//		Identity types.String `tfsdk:"identity"`
//	}
type linuxHostProvider struct {
	version string
	client  *linuxhost_client.SSHClientContext
}

var _ provider.Provider = &linuxHostProvider{}

func (p *linuxHostProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config struct {
		Host       string  `tfsdk:"host"`
		Username   string  `tfsdk:"username"`
		Password   *string `tfsdk:"password"`
		PrivateKey *string `tfsdk:"private_key"`
		Port       *int64  `tfsdk:"port"`
	}

	diags := req.Config.Get(ctx, &config)
	tflog.Info(ctx, "Configuring linuxhost PROVIDER")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle defaults for optional values
	if config.Port == nil {
		// Assign a default value for Port if it is not set
		defaultPort := int64(22)
		config.Port = &defaultPort
	}

	if config.PrivateKey == nil && config.Password == nil {

		resp.Diagnostics.AddError(
			"Missing authentication method",
			"Either 'password' or 'private_key' must be provided for SSH authentication.",
		)
		return
	}
	if config.PrivateKey == nil {
		empty := ""
		config.PrivateKey = &empty
	}
	if config.Password == nil {
		empty := ""
		config.Password = &empty
	}

	clientContext, err := linuxhost_client.NewSSHClient(config.Host, *config.Port, config.Username, *config.Password, *config.PrivateKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to connect SSH client",
			fmt.Sprintf("Error connecting SSH client: %s", err),
		)
		return
	}
	tflog.Info(ctx, "Should now be connected")

	hostData := &linuxhost_client.HostData{
		Client: clientContext,
	}

	resp.DataSourceData = hostData
	resp.ResourceData = hostData

	p.client = clientContext
}

func (p *linuxHostProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *linuxHostProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "linuxhost"
	resp.Version = p.version
}

func (p *linuxHostProvider) Resources(ctx context.Context) []func() resource.Resource {
	tflog.Info(ctx, "Configuring resources ")
	return []func() resource.Resource{
		NewNetworkInterfaceResource,
		NewNetworkInterfaceIPResource,
		NewUserResource,
		NewGroupResource,
		NewCaCertificateResource,
		NewIfBridgeResource,
		NewIfVethResource,
		NewIfVxlanResource,
	}
}

func (p *linuxHostProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The SSH hostname or IP address to connect to.",
				Required:    true,
			},
			"username": schema.StringAttribute{
				Description: "The SSH username.",
				Required:    true,
			},
			"password": schema.StringAttribute{
				Description: "The SSH password (if not using a private key).",
				Optional:    true,
				Sensitive:   true,
			},
			"private_key": schema.StringAttribute{
				Description: "The private key for SSH authentication.",
				Optional:    true,
				Sensitive:   true,
			},
			"port": schema.Int64Attribute{
				Description: "The SSH port to connect to.",
				Optional:    true,
			},
		},
	}
}

func New(version string) func() provider.Provider {
	// fmt.Println("Configuring HashiCups client")
	return func() provider.Provider {
		return &linuxHostProvider{
			version: version,
		}
	}
}

// // ///////////////
// // ScaffoldingProvider defines the provider implementation.
// type ScaffoldingProvider struct {
// 	// version is set to the provider version on release, "dev" when the
// 	// provider is built and ran locally, and "test" when running acceptance
// 	// testing.
// 	version string
// }

// // ScaffoldingProviderModel describes the provider data model.
// type ScaffoldingProviderModel struct {
// 	Endpoint types.String `tfsdk:"endpoint"`
// }

// func (p *ScaffoldingProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
// 	resp.TypeName = "scaffolding"
// 	resp.Version = p.version
// }

// func (p *ScaffoldingProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
// 	resp.Schema = schema.Schema{
// 		Attributes: map[string]schema.Attribute{
// 			"endpoint": schema.StringAttribute{
// 				MarkdownDescription: "Example provider attribute",
// 				Optional:            true,
// 			},
// 		},
// 	}
// }

// func (p *ScaffoldingProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
// 	var data ScaffoldingProviderModel

// 	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

// 	if resp.Diagnostics.HasError() {
// 		return
// 	}

// 	// Configuration values are now available.
// 	// if data.Endpoint.IsNull() { /* ... */ }

// 	// Example client configuration for data sources and resources
// 	client := http.DefaultClient
// 	resp.DataSourceData = client
// 	resp.ResourceData = client
// }

// func (p *ScaffoldingProvider) Resources(ctx context.Context) []func() resource.Resource {
// 	return []func() resource.Resource{
// 		NewExampleResource,
// 	}
// }

// func (p *ScaffoldingProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
// 	return []func() datasource.DataSource{
// 		NewExampleDataSource,
// 	}
// }

// func (p *ScaffoldingProvider) Functions(ctx context.Context) []func() function.Function {
// 	return []func() function.Function{
// 		NewExampleFunction,
// 	}
// }

// func New(version string) func() provider.Provider {
// 	return func() provider.Provider {
// 		return &ScaffoldingProvider{
// 			version: version,
// 		}
// 	}
// }
