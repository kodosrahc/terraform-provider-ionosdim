// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"strings"

	"terraform-provider-ionosdim/pkg/dim"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure ionosdimProvider satisfies various provider interfaces.
var _ provider.Provider = &ionosdimProvider{}

// ionosdimProvider defines the provider implementation.
type ionosdimProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ionosdimProviderModel describes the provider data model.
type ionosdimProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Token    types.String `tfsdk:"token"`
}

func (p *ionosdimProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ionosdim"
	resp.Version = p.version
}

func (p *ionosdimProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "DIM endpoint, e.g. https://dim.example.com/dim . Can be sourced from `IONOSDIM_ENDPOINT` environment variable.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "DIM username, it is ignored if `token` is specified. Can be sourced from `IONOSDIM_USERNAME` environment variable.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "DIM user password, it is ignored if `token` is specified. Can be sourced from `IONOSDIM_PASSWORD` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "DIM token. Can be sourced from `IONOSDIM_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *ionosdimProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ionosdimProviderModel

	tflog.Info(ctx, "Configuring IonosDim Provider")

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// if data.Endpoint.IsNull() { /* ... */ }
	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown IonosDim API Endpoint",
			"The provider cannot create the IonosDim API client as there is an unknown configuration value for the IonosDim API endpoint. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the IONOSDIM_ENDPOINT environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown IonosDim API Username",
			"The provider cannot create the IonosDim API client as there is an unknown configuration value for the IonosDim API username. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the IONOSDIM_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown IonosDim API Password",
			"The provider cannot create the IonosDim API client as there is an unknown configuration value for the IonosDim API password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the IONOSDIM_PASSWORD environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown IonosDim API Token",
			"The provider cannot create the IonosDim API client as there is an unknown configuration value for the IonosDim API token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the IONOSDIM_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	endpoint := os.Getenv("IONOSDIM_ENDPOINT")
	username := os.Getenv("IONOSDIM_USERNAME")
	password := os.Getenv("IONOSDIM_PASSWORD")
	token := os.Getenv("IONOSDIM_TOKEN")

	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	endpoint = strings.TrimRight(endpoint, "/")
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing IonosDim API Endpoint",
			"The provider cannot create the IonosDim API client as there is a missing or empty value for the IonosDim API endpoint. "+
				"Set the endpoint value in the configuration or use the IONOSDIM_ENDPOINT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		if username == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Missing IonosDim API Username",
				"The provider cannot create the IonosDim API client as there is a missing or empty value for the IonosDim API username. "+
					"Set the username value in the configuration or use the IONOSDIM_USERNAME environment variable. "+
					"If either is already set, ensure the value is not empty. "+
					"Alternatively, set the token value in the configuration or use the IONOSDIM_TOKEN environment variable.",
			)
		}

		if password == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Missing IonosDim API Password",
				"The provider cannot create the IonosDim API client as there is a missing or empty value for the IonosDim API password. "+
					"Set the password value in the configuration or use the IONOSDIM_PASSWORD environment variable. "+
					"If either is already set, ensure the value is not empty. "+
					"Alternatively, set the token value in the configuration or use the IONOSDIM_TOKEN environment variable.",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "endpoint", endpoint)
	ctx = tflog.SetField(ctx, "username", username)
	ctx = tflog.SetField(ctx, "password", password)
	ctx = tflog.SetField(ctx, "token", token)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "password", "token")
	tflog.Debug(ctx, "Creating IonosDim API Client")
	// Create a new HashiCups client using the configuration values
	client, err := dim.NewClientWithContext(ctx, &endpoint, &token, &username, &password, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create IonosDim API Client",
			"An unexpected error occurred when creating the IonosDim API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"IonosDim Client Error: "+err.Error(),
		)
		return
	}

	// Make the HashiCups client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured IonosDim API client", map[string]any{"success": true})
}

func (p *ionosdimProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewIpResource,
		NewARecordResource,
		NewCNAMERecordResource,
		NewTXTRecordResource,
	}
}

func (p *ionosdimProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewARecordSetDataSource,
		NewCNAMERecordSetDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ionosdimProvider{
			version: version,
		}
	}
}
