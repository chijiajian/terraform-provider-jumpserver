package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &accountResource{}

// 资源结构体
type accountResource struct {
	client *http.Client
}

type JumpServerAccountModel struct {
	//	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`       // 必填
	Username   types.String `tfsdk:"username"`   // 必填
	Privileged types.Bool   `tfsdk:"privileged"` // 必填
	Is_active  types.Bool   `tfsdk:"is_active"`  // 必填
	Assets     types.List   `tfsdk:"assets"`     // 必填
}

func AccountResource() resource.Resource {
	return &accountResource{}
}

func (r *accountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (r *accountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *accountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the account",
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username of the account",
			},
			"privileged": schema.BoolAttribute{
				Required:    true,
				Description: "The platform of the asset host",
			},
			"is_active": schema.BoolAttribute{
				Required:    true,
				Description: "The nodes display of the asset host",
			},
			"assets": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// 创建资源
func (r *accountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan JumpServerAccountModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var validAssets []string
	for _, asset := range plan.Assets.Elements() {
		assetStr := asset.String()
		// 去除两侧的引号或额外字符
		assetStr = strings.Trim(assetStr, `“”"`)

		// 验证 UUID 格式
		if _, err := uuid.Parse(assetStr); err != nil {
			resp.Diagnostics.AddError("Invalid UUID", fmt.Sprintf("Asset '%s' is not a valid UUID", assetStr))
			return
		}
		validAssets = append(validAssets, assetStr)
	}
	// 构建请求体
	payload := map[string]interface{}{
		"name":       plan.Name.ValueString(),
		"username":   plan.Username.ValueString(),
		"privileged": plan.Privileged.ValueBool(),
		"is_active":  plan.Is_active.ValueBool(),
		"assets":     validAssets,
	}

	// 将请求体转换为 JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request data", err.Error())
		return
	}

	url := "http://172.30.9.65/api/v1/accounts/accounts/bulk/"
	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HTTP request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送 HTTP 请求
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error sending HTTP request", err.Error())
		return
	}
	defer httpResp.Body.Close()

	// 检查响应状态码
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unexpected status code: %d, Response: %s", httpResp.StatusCode, string(body)))
		return
	}

	// 从 API 响应中解析资源 ID（假设返回一个 ID 字段）
	// 解析 API 响应
	var apiResponse []map[string]interface{}
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResponse); err != nil {
		resp.Diagnostics.AddError("Error decoding API response", err.Error())
		return
	}
	// 假设 API 响应为 [{"asset":"jumperServer(172.30.9.65)","state":"created","changed":true}]
	if len(apiResponse) > 0 {
		assetInfo := apiResponse[0]

		// 如果创建成功，并且可以从响应中获取 asset 字段
		if state, ok := assetInfo["state"].(string); ok && state == "created" {
			// 在这里，可以选择记录状态、设置资源的其他属性
			// 例如，将 "asset" 赋值给模型字段（可以忽略 SetId）
			// 或者记录日志等
		} else {
			resp.Diagnostics.AddError("Invalid API response", "The response did not contain the expected data")
			return
		}
	}

	// 更新 Terraform 状态
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// 读取资源
func (r *accountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

}

// 更新资源
func (r *accountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

}

// 删除资源
func (r *accountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

}
