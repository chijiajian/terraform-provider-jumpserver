package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &assetHostResource{}

// 资源结构体
type assetHostResource struct {
	client *http.Client
}

func AssetHostResource() resource.Resource {
	return &assetHostResource{}
}

type JumpServerHostResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`          // 必填
	IP           types.String `tfsdk:"ip"`            // 必填
	Platform     types.String `tfsdk:"platform"`      // 必填
	NodesDisplay types.List   `tfsdk:"nodes_display"` // 必填
	Protocols    types.List   `tfsdk:"protocols"`     // 必填
}

// 协议数据模型
type ProtocolModel struct {
	Name types.String `tfsdk:"name"` // 必填
	Port types.Int64  `tfsdk:"port"` // 可选
}

func (r *assetHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_asset_host"
}

func (r *assetHostResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

func (r *assetHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the asset host",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the asset host",
			},
			"ip": schema.StringAttribute{
				Required:    true,
				Description: "The IP address of the asset host",
			},
			"platform": schema.StringAttribute{
				Required:    true,
				Description: "The platform of the asset host",
			},
			"nodes_display": schema.ListAttribute{
				Required:    true,
				Description: "The nodes display of the asset host",
				ElementType: types.StringType,
			},
			"protocols": schema.ListNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
						},
						"port": schema.Int64Attribute{
							Optional: true,
						},
					},
				},
			},
		},
	}
}

// 创建资源
func (r *assetHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan JumpServerHostResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 解析用户定义的协议
	var protocols []map[string]interface{}
	for _, proto := range plan.Protocols.Elements() {
		protoObj, ok := proto.(types.Object)
		if !ok {
			resp.Diagnostics.AddError("Type Assertion Error", "Failed to assert protocol as types.Object")
			return
		}

		nameAttr, nameOk := protoObj.Attributes()["name"]
		portAttr, portOk := protoObj.Attributes()["port"]

		if !nameOk {
			resp.Diagnostics.AddError("Missing Attribute", "Protocol name is required")
			return
		}

		protocol := map[string]interface{}{
			"name": nameAttr.(types.String).ValueString(),
		}

		if portOk {
			protocol["port"] = portAttr.(types.Int64).ValueInt64()
		}

		protocols = append(protocols, protocol)
	}

	var nodesDisplay []string
	if !plan.NodesDisplay.IsNull() {
		var nodes []types.String
		diags := plan.NodesDisplay.ElementsAs(context.Background(), &nodes, false)
		if diags.HasError() {
			resp.Diagnostics.AddError("Data Conversion Error", "Failed to convert nodes_display to []string")
			return
		}
		for _, node := range nodes {
			nodesDisplay = append(nodesDisplay, node.ValueString())
		}
	}

	// 构造请求体
	asset := map[string]interface{}{
		"name":          plan.Name.ValueString(),     // 使用 "name"
		"address":       plan.IP.ValueString(),       // 使用 "address"
		"platform":      plan.Platform.ValueString(), //1,                       // 使用整数形式的平台 ID
		"nodes_display": nodesDisplay,                // 使用 "nodes_display"
		"protocols":     protocols,
		"is_active":     true, // 默认激活
	}

	apiPath := "/api/v1/assets/hosts/" // 确保路径包含 API 版本
	fullURL := fmt.Sprintf("%s%s", r.client.Transport.(*authTransport).BaseURL, apiPath)

	jsonValue, err := json.Marshal(asset) // 直接传递 asset，不需要包装在 "data" 字段中
	if err != nil {
		resp.Diagnostics.AddError("JSON Marshal Error", fmt.Sprintf("Error marshaling request body: %v", err))
		return
	}

	reqBody := bytes.NewBuffer(jsonValue)

	// 打印调试信息
	fmt.Println("Full URL:", fullURL)
	fmt.Println("Request Body:", string(jsonValue))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, reqBody) // 确保使用 POST 方法
	if err != nil {
		resp.Diagnostics.AddError("HTTP Request Error", fmt.Sprintf("Error creating asset: %v", err))
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.client.Transport.(*authTransport).Token)

	client := &http.Client{}
	respBody, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Request Error", fmt.Sprintf("Error creating asset: %v", err))
		return
	}
	defer respBody.Body.Close()

	// 打印响应状态码和响应体
	body, _ := io.ReadAll(respBody.Body)
	fmt.Println("Response Status:", respBody.Status)
	fmt.Println("Response Body:", string(body))

	if respBody.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError("HTTP Status Error", fmt.Sprintf("Error creating asset: %s, Response: %s", respBody.Status, string(body)))
		return
	}

	// 解析响应体
	var result map[string]interface{}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Response Decode Error", fmt.Sprintf("Error decoding response: %v", err))
		return
	}

	// 提取资产的 ID
	if id, ok := result["id"].(string); ok {
		plan.ID = types.StringValue(id)
	} else {
		resp.Diagnostics.AddError("API Error", "Unable to retrieve asset ID from response")
		return
	}

	// 更新 Terraform 状态
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// 读取资源
func (r *assetHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state JumpServerHostResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	apiPath := fmt.Sprintf("/api/v1/assets/hosts/%s/", id)
	fullURL := fmt.Sprintf("%s%s", r.client.Transport.(*authTransport).BaseURL, apiPath)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Request Error", fmt.Sprintf("Unable to create request: %s", err))
		return
	}
	httpReq.Header.Set("accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.client.Transport.(*authTransport).Token)

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Request Error", fmt.Sprintf("Unable to send request: %s", err))
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status code: %s, Response: %s", httpResp.Status, string(body)))
		return
	}

	// 适配 API 返回的对象
	var result map[string]interface{}
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", fmt.Sprintf("Unable to decode response: %s", err))
		return
	}

	// 更新状态
	if name, ok := result["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if ip, ok := result["ip"].(string); ok {
		state.IP = types.StringValue(ip)
	}
	if platform, ok := result["platform"].(string); ok {
		state.Platform = types.StringValue(platform)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

/*
func (r *assetHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state JumpServerHostResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 发送请求
	id := state.ID.ValueString()

	apiPath := "/api/v1/assets/hosts/suggestions/"
	queryParams := url.Values{}

	queryParams.Add("id", id)

	fullURL := fmt.Sprintf("%s%s?%s", r.client.Transport.(*authTransport).BaseURL, apiPath, queryParams.Encode())

	httpReq, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Error", fmt.Sprintf("Unable to create request: %s", err))
		return
	}

	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Error", fmt.Sprintf("Unable to send request: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// 检查响应状态码
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status code: %s", httpResp.Status))
		return
	}

	var response struct {
		Count    int                      `json:"count"`
		Next     string                   `json:"next"`
		Previous string                   `json:"previous"`
		Results  []map[string]interface{} `json:"results"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&response); err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", fmt.Sprintf("Unable to decode response: %s", err))
		return
	}

	// 检查 results 是否为空
	if len(response.Results) == 0 {
		resp.Diagnostics.AddError("API Error", "No results found for the given ID")
		return
	}

	result := response.Results[0]
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("JSON Decode Error", fmt.Sprintf("Unable to decode response: %s", err))
		return
	}

	state.Name = types.StringValue(result["name"].(string))
	state.IP = types.StringValue(result["ip"].(string))
	state.Platform = types.StringValue(result["platform"].(string))
	protocols := result["protocols"].([]interface{})
	protocolsList, diags := types.ListValueFrom(ctx, types.StringType, protocols)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Protocols = protocolsList
	nodesDisplay := result["nodes_display"].([]interface{})
	nodesDisplayList, diags := types.ListValueFrom(ctx, types.StringType, nodesDisplay)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.NodesDisplay = nodesDisplayList

	// 保存状态
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
*/
// 更新资源
func (r *assetHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

}

// 删除资源
func (r *assetHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state JumpServerHostResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// 获取资源 ID
	id := state.ID.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Resource ID is required for deletion")
		return
	}

	// 构造 API URL
	apiPath := fmt.Sprintf("/api/v1/assets/hosts/%s/", id)
	fullURL := fmt.Sprintf("%s%s", r.client.Transport.(*authTransport).BaseURL, apiPath)

	// 创建 HTTP DELETE 请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, fullURL, nil)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Request Error", fmt.Sprintf("Unable to create request: %s", err))
		return
	}

	// 设置请求头
	httpReq.Header.Set("accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+r.client.Transport.(*authTransport).Token)

	// 发送请求
	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Request Error", fmt.Sprintf("Unable to send request: %s", err))
		return
	}
	defer httpResp.Body.Close()

	// 检查响应状态码
	if httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status code: %s, Response: %s", httpResp.Status, string(body)))
		return
	}

	// 标记资源为已删除
	resp.State.RemoveResource(ctx)
}
