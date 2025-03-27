// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure Netbird satisfies various provider interfaces.
var _ provider.Provider = &NetbirdProvider{}
var _ provider.ProviderWithFunctions = &NetbirdProvider{}
var _ provider.ProviderWithEphemeralResources = &NetbirdProvider{}

// NetbirdProvider defines the provider implementation.
type NetbirdProvider struct {
	version string
}

// NetbirdProviderModel describes the provider data model.
type NetbirdProviderModel struct {
	Endpoint    types.String `tfsdk:"endpoint"`
	BearerToken types.String `tfsdk:"bearer_token"`
	AccessToken types.String `tfsdk:"access_token"`
}

func (p *NetbirdProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "netbird"
	resp.Version = p.version
}

func (p *NetbirdProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "HTTPS endpoint to netbird API. Defaults to `api.netbird.io`.",
				Optional:            true,
			},
			"bearer_token": schema.StringAttribute{
				MarkdownDescription: "Oauth2 Bearer Token",
				Optional:            true,
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "PAT (personal access token)",
				Optional:            true,
			},
		},
	}
}

func (p *NetbirdProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data NetbirdProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	bearerToken := os.Getenv("NETBIRD_BEARER_TOKEN")
	accessToken := os.Getenv(("NETBIRD_ACCESS_TOKEN"))
	endpoint := os.Getenv("NETBIRD_ENDPOINT")

	// Configuration values are now available.
	if data.Endpoint.ValueString() != "" {
		endpoint = data.Endpoint.ValueString()
	}

	if endpoint == "" {
		endpoint = "https://api.netbird.io"
	}

	if providerBearerString := data.BearerToken.ValueString(); providerBearerString != "" {
		bearerToken = providerBearerString
	}

	if providerAccessToken := data.AccessToken.ValueString(); providerAccessToken != "" {
		accessToken = providerAccessToken
	}

	if bearerToken == "" && accessToken == "" {
		resp.Diagnostics.AddError(
			"Bearer token and access token missing.",
			"The provider must be configured with either the `bearer_token` or the `access_token` to authenticate to Netbird. "+
				"Set one of these values in the configuration. "+
				"If either is already set, ensure the value is not empty. "+
				"If this was not expected, please check for NETBIRD_* environment variables. "+
				"See the provider documentation for more information",
		)
	}
	if bearerToken != "" && accessToken != "" {
		resp.Diagnostics.AddError(
			"Conflicting arguments: Bearer token and access token.",
			"The provider must be configured with either the `bearer_token` or the `access_token` to authenticate to Netbird. "+
				"Only set one of these values in the configuration. "+
				"If this was not expected, please check for NETBIRD_* environment variables. "+
				"See the provider documentation for more information",
		)
	}

	// Example client configuration for data sources and resources
	client := NewClient(endpoint, bearerToken, accessToken)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *NetbirdProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNetworkResource,
		NewGroupResource,
		NewPolicyResource,
		NewNetworkRouterResource,
		NewNetworkResourceResource,
		NewNameserverGroupResource,
	}
}

func (p *NetbirdProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		// NewExampleEphemeralResource,
	}
}

func (p *NetbirdProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewPeersDataSource,
		NewPeerDataSource,
	}
}

func (p *NetbirdProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		// NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NetbirdProvider{
			version: version,
		}
	}
}
