package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the data source implements the required interfaces.
var _ datasource.DataSource = &HostSuggestionsDataSource{}

// HostSuggestionsDataSource defines the data source implementation.
type HostSuggestionsDataSource struct {
	client *http.Client
}

// HostSuggestionsDataSourceModel describes the data source data model.
type HostSuggestionsDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Address               types.String `tfsdk:"address"`
	IsActive              types.String `tfsdk:"is_active"`
	Type                  types.String `tfsdk:"type"`
	Category              types.String `tfsdk:"category"`
	Platform              types.String `tfsdk:"platform"`
	IsGateway             types.String `tfsdk:"is_gateway"`
	ExcludePlatform       types.String `tfsdk:"exclude_platform"`
	Domain                types.String `tfsdk:"domain"`
	Protocols             types.String `tfsdk:"protocols"`
	DomainEnabled         types.String `tfsdk:"domain_enabled"`
	PingEnabled           types.String `tfsdk:"ping_enabled"`
	GatherFactsEnabled    types.String `tfsdk:"gather_facts_enabled"`
	ChangeSecretEnabled   types.String `tfsdk:"change_secret_enabled"`
	PushAccountEnabled    types.String `tfsdk:"push_account_enabled"`
	VerifyAccountEnabled  types.String `tfsdk:"verify_account_enabled"`
	GatherAccountsEnabled types.String `tfsdk:"gather_accounts_enabled"`
	Search                types.String `tfsdk:"search"`
	Order                 types.String `tfsdk:"order"`
	Limit                 types.Int64  `tfsdk:"limit"`
	Offset                types.Int64  `tfsdk:"offset"`
	Results               []HostModel  `tfsdk:"results"`
	TotalCount            types.Int64  `tfsdk:"total_count"`
	Next                  types.String `tfsdk:"next"`
	Previous              types.String `tfsdk:"previous"`
}

// HostModel describes a single host result.
type HostModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	// Add other fields as needed based on the Host schema in the API.
}

func NewHostSuggestionsDataSource() datasource.DataSource {
	return &HostSuggestionsDataSource{}
}

func (d *HostSuggestionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_suggestions"
}

func (d *HostSuggestionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches host suggestions from JumpServer based on query parameters.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the host.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the host.",
				Optional:    true,
			},
			"address": schema.StringAttribute{
				Description: "The address of the host.",
				Optional:    true,
			},
			"is_active": schema.StringAttribute{
				Description: "Whether the host is active.",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "The type of the host.",
				Optional:    true,
			},
			"category": schema.StringAttribute{
				Description: "The category of the host.",
				Optional:    true,
			},
			"platform": schema.StringAttribute{
				Description: "The platform of the host.",
				Optional:    true,
			},
			"is_gateway": schema.StringAttribute{
				Description: "Whether the host is a gateway.",
				Optional:    true,
			},
			"exclude_platform": schema.StringAttribute{
				Description: "Exclude hosts with this platform.",
				Optional:    true,
			},
			"domain": schema.StringAttribute{
				Description: "The domain of the host.",
				Optional:    true,
			},
			"protocols": schema.StringAttribute{
				Description: "The protocols supported by the host.",
				Optional:    true,
			},
			"domain_enabled": schema.StringAttribute{
				Description: "Whether the domain is enabled.",
				Optional:    true,
			},
			"ping_enabled": schema.StringAttribute{
				Description: "Whether ping is enabled.",
				Optional:    true,
			},
			"gather_facts_enabled": schema.StringAttribute{
				Description: "Whether gathering facts is enabled.",
				Optional:    true,
			},
			"change_secret_enabled": schema.StringAttribute{
				Description: "Whether changing secrets is enabled.",
				Optional:    true,
			},
			"push_account_enabled": schema.StringAttribute{
				Description: "Whether pushing accounts is enabled.",
				Optional:    true,
			},
			"verify_account_enabled": schema.StringAttribute{
				Description: "Whether verifying accounts is enabled.",
				Optional:    true,
			},
			"gather_accounts_enabled": schema.StringAttribute{
				Description: "Whether gathering accounts is enabled.",
				Optional:    true,
			},
			"search": schema.StringAttribute{
				Description: "A search term.",
				Optional:    true,
			},
			"order": schema.StringAttribute{
				Description: "The field to use when ordering the results.",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "The number of results to return per page.",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "The initial index from which to return the results.",
				Optional:    true,
			},
			"results": schema.ListNestedAttribute{
				Description: "The list of host suggestions.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the host.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the host.",
							Computed:    true,
						},
						// Add other fields as needed.
					},
				},
			},
			"total_count": schema.Int64Attribute{
				Description: "The total number of results.",
				Computed:    true,
			},
			"next": schema.StringAttribute{
				Description: "The URL for the next page of results.",
				Computed:    true,
			},
			"previous": schema.StringAttribute{
				Description: "The URL for the previous page of results.",
				Computed:    true,
			},
		},
	}
}

func (d *HostSuggestionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*http.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *HostSuggestionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data HostSuggestionsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build query parameters
	queryParams := url.Values{}
	if !data.ID.IsNull() {
		queryParams.Add("id", data.ID.ValueString())
	}
	if !data.Name.IsNull() {
		queryParams.Add("name", data.Name.ValueString())
	}
	if !data.Address.IsNull() {
		queryParams.Add("address", data.Address.ValueString())
	}
	if !data.IsActive.IsNull() {
		queryParams.Add("is_active", data.IsActive.ValueString())
	}
	if !data.Type.IsNull() {
		queryParams.Add("type", data.Type.ValueString())
	}
	if !data.Category.IsNull() {
		queryParams.Add("category", data.Category.ValueString())
	}
	if !data.Platform.IsNull() {
		queryParams.Add("platform", data.Platform.ValueString())
	}
	if !data.IsGateway.IsNull() {
		queryParams.Add("is_gateway", data.IsGateway.ValueString())
	}
	if !data.ExcludePlatform.IsNull() {
		queryParams.Add("exclude_platform", data.ExcludePlatform.ValueString())
	}
	if !data.Domain.IsNull() {
		queryParams.Add("domain", data.Domain.ValueString())
	}
	if !data.Protocols.IsNull() {
		queryParams.Add("protocols", data.Protocols.ValueString())
	}
	if !data.DomainEnabled.IsNull() {
		queryParams.Add("domain_enabled", data.DomainEnabled.ValueString())
	}
	if !data.PingEnabled.IsNull() {
		queryParams.Add("ping_enabled", data.PingEnabled.ValueString())
	}
	if !data.GatherFactsEnabled.IsNull() {
		queryParams.Add("gather_facts_enabled", data.GatherFactsEnabled.ValueString())
	}
	if !data.ChangeSecretEnabled.IsNull() {
		queryParams.Add("change_secret_enabled", data.ChangeSecretEnabled.ValueString())
	}
	if !data.PushAccountEnabled.IsNull() {
		queryParams.Add("push_account_enabled", data.PushAccountEnabled.ValueString())
	}
	if !data.VerifyAccountEnabled.IsNull() {
		queryParams.Add("verify_account_enabled", data.VerifyAccountEnabled.ValueString())
	}
	if !data.GatherAccountsEnabled.IsNull() {
		queryParams.Add("gather_accounts_enabled", data.GatherAccountsEnabled.ValueString())
	}
	if !data.Search.IsNull() {
		queryParams.Add("search", data.Search.ValueString())
	}
	if !data.Order.IsNull() {
		queryParams.Add("order", data.Order.ValueString())
	}
	if !data.Limit.IsNull() {
		queryParams.Add("limit", fmt.Sprintf("%d", data.Limit.ValueInt64()))
	}
	if !data.Offset.IsNull() {
		queryParams.Add("offset", fmt.Sprintf("%d", data.Offset.ValueInt64()))
	}

	// Build the full URL with query parameters
	apiPath := "/api/v1/assets/hosts/suggestions/"
	fullURL := fmt.Sprintf("%s%s?%s", d.client.Transport.(*authTransport).BaseURL, apiPath, queryParams.Encode())

	// Send the HTTP GET request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create HTTP request",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	httpResp, err := d.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to send HTTP request",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}
	defer httpResp.Body.Close()

	// Check for a successful response
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected HTTP response status",
			fmt.Sprintf("Received status code: %d", httpResp.StatusCode),
		)
		return
	}

	// Parse the JSON response
	var apiResponse []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		// Add other fields as needed based on the API response
	}

	if err := json.NewDecoder(httpResp.Body).Decode(&apiResponse); err != nil {
		resp.Diagnostics.AddError(
			"Failed to decode JSON response",
			fmt.Sprintf("Error: %s", err),
		)
		return
	}

	// Map the API response to the Terraform data model
	data.TotalCount = types.Int64Value(int64(len(apiResponse)))
	data.Next = types.StringNull()     // If the API does not provide pagination info, set to null
	data.Previous = types.StringNull() // If the API does not provide pagination info, set to null

	data.Results = make([]HostModel, 0, len(apiResponse))
	for _, result := range apiResponse {
		data.Results = append(data.Results, HostModel{
			ID:   types.StringValue(result.ID),
			Name: types.StringValue(result.Name),
			// Map other fields as needed
		})
	}

	// Set the data model as the response
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
