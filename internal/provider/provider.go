// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure JumpServerProvider satisfies various provider interfaces.
var _ provider.Provider = &JumpServerProvider{}

// JumpServerProvider defines the provider implementation.
type JumpServerProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// JumpServerProviderModel describes the provider data model.
type JumpServerProviderModel struct {
	BaseURL  types.String `tfsdk:"base_url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Token    types.String `tfsdk:"token"`
}

func (p *JumpServerProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "jumpserver"
	resp.Version = p.version
}

func (p *JumpServerProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "The base URL of the JumpServer API",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username for authentication",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for authentication",
				Required:            true,
				Sensitive:           true,
			},
			"token": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (p *JumpServerProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data JumpServerProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := os.Getenv("JUMP_SERVER_BASE_URL")
	username := os.Getenv("JUMP_SERVER_USERNAME")
	password := os.Getenv("JUMP_SERVER_PASSWORD")

	if !data.BaseURL.IsNull() {
		baseURL = data.BaseURL.ValueString()
	}
	if !data.Username.IsNull() {
		username = data.Username.ValueString()
	}
	if !data.Password.IsNull() {
		password = data.Password.ValueString()
	}

	if baseURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Missing JumpServer API Base URL",
			"The provider cannot create the JumpServer API client as there is a missing or empty value for the JumpServer API base URL. "+
				"Set the base_url value in the configuration or use the JUMP_SERVER_BASE_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing JumpServer API Username",
			"The provider cannot create the JumpServer API client as there is a missing or empty value for the JumpServer API username. "+
				"Set the username value in the configuration or use the JUMP_SERVER_USERNAME environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing JumpServer API Password",
			"The provider cannot create the JumpServer API client as there is a missing or empty value for the JumpServer API password. "+
				"Set the password value in the configuration or use the JUMP_SERVER_PASSWORD environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	token, err := getToken(baseURL, username, password)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to authenticate with JumpServer API",
			fmt.Sprintf("An unexpected error occurred when trying to authenticate with the JumpServer API: %s", err.Error()),
		)
		return
	}

	client := &http.Client{}
	client.Transport = &authTransport{
		Token:    token,
		BaseURL:  baseURL,
		Delegate: http.DefaultTransport,
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func getToken(baseURL, username, password string) (string, error) {
	url := baseURL + "/api/v1/authentication/auth/"
	credentials := map[string]string{
		"username": username,
		"password": password,
	}
	jsonValue, _ := json.Marshal(credentials)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if token, ok := result["token"].(string); ok {
		return token, nil
	}
	return "", fmt.Errorf("unable to fetch token")
}

type authTransport struct {
	Token    string
	BaseURL  string
	Delegate http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	return t.Delegate.RoundTrip(req)
}

func (p *JumpServerProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		AssetHostResource,
		AccountResource,
	}
}

func (p *JumpServerProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHostSuggestionsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &JumpServerProvider{
			version: version,
		}
	}
}
