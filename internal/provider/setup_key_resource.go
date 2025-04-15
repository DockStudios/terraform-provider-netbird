package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	netbirdApi "github.com/netbirdio/netbird/management/server/http/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SetupKeyResource{}
var _ resource.ResourceWithImportState = &SetupKeyResource{}

func NewSetupKeyResource() resource.Resource {
	return &SetupKeyResource{}
}

// SetupKeyResource defines the resource implementation.
type SetupKeyResource struct {
	client *Client
}

type SetupKeyResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Type                types.String `tfsdk:"type"`
	AutoGroups          types.List   `tfsdk:"auto_group"`
	ExpiresIn           types.Int64  `tfsdk:"expires_in"`
	UsageLimit          types.Int64  `tfsdk:"usage_limit"`
	AllowExtraDnsLabels types.Bool   `tfsdk:"allow_extra_dns_labels"`
	Ephemeral           types.Bool   `tfsdk:"ephemeral"`
}

func (r *SetupKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setup_key"
}

func (r *SetupKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Setup Key resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Setup Key ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Setup Key Name",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Setup key type, `one-off` for single time usage and `reusable`",
				Optional:            true,
			},
			"expires_in": schema.Int64Attribute{
				MarkdownDescription: "Expiration time in seconds",
				Computed:            true,
				Required:            true,
			},
			"usage_limit": schema.Int64Attribute{
				MarkdownDescription: "A number of times this key can be used. The value of 0 indicates the unlimited usage.",
				Required:            true,
			},
			"ephemeral": schema.BoolAttribute{
				MarkdownDescription: "Indicate that the peer will be ephemeral or not",
				Optional:            true,
			},
			"allow_extra_dns_labels": schema.BoolAttribute{
				MarkdownDescription: "Allow extra DNS labels to be added to the peer",
				Optional:            true,
			},
			"auto_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of group IDs to auto-assign to peers registered with this key",
				Computed:            true,
			},
		},
	}
}

func (r *SetupKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *SetupKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SetupKeyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	requestBody, err := json.Marshal(map[string]string{
		"name":        data.Name.ValueString(),
		"description": data.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	// Make API request
	reqURL := fmt.Sprintf("%s/api/networks", r.client.BaseUrl)
	httpReq, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(requestBody))
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	responseBody, err := r.client.doRequest(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error making API request", err.Error())
		return
	}

	// Parse response
	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Assign values from API response
	data.ID = types.StringValue(responseData["id"].(string))

	diags := r.readIntoModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SetupKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SetupKeyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.readIntoModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SetupKeyResource) readIntoModel(ctx context.Context, data *SetupKeyResourceModel) diag.Diagnostics {
	// Update network model
	// Fetch data from API
	diags := diag.Diagnostics{}
	reqURL := fmt.Sprintf("%s/api/networks/%s", r.client.BaseUrl, data.ID.ValueString())
	httpReq, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		diags.AddError("Error creating request", err.Error())
		return diags
	}

	responseBody, err := r.client.doRequest(httpReq)
	if err != nil {
		diags.AddError("Error fetching network", err.Error())
		return diags
	}

	// Handle when resource does not exist
	if responseBody == nil {
		data.ID = types.StringNull()
		return diags
	}

	var responseData netbirdApi.Network
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		diags.AddError("Error parsing response", err.Error())
		return diags
	}
	// Update state with latest data
	data.Name = types.StringValue(responseData.Name)

	// Only update if either (or both) data and response data have a non-empty description value
	if (responseData.Description != nil && (*responseData.Description) != string("")) || data.Description.ValueString() != "" {
		if responseData.Description != nil {
			data.Description = types.StringValue(*responseData.Description)
		} else {
			responseData.Description = types.StringNull().ValueStringPointer()
		}
	}
	data.RoutingPeersCount = types.Int64Value(int64(responseData.RoutingPeersCount))

	routers := responseData.Routers
	routersModel, newDiags := types.ListValueFrom(ctx, types.StringType, routers)
	diags.Append(newDiags...)
	data.Routers = routersModel

	resources := responseData.Resources
	data.Resources, newDiags = types.ListValueFrom(ctx, types.StringType, resources)
	diags.Append(newDiags...)

	policies := responseData.Policies
	data.Policies, newDiags = types.ListValueFrom(ctx, types.StringType, policies)
	diags.Append(newDiags...)

	return diags
}

func (r *SetupKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SetupKeyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	requestBody, err := json.Marshal(map[string]string{
		"name":        data.Name.ValueString(),
		"description": data.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	reqURL := fmt.Sprintf("%s/api/networks/%s", r.client.BaseUrl, data.ID.ValueString())
	httpReq, err := http.NewRequest("PUT", reqURL, bytes.NewBuffer(requestBody))
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	_, err = r.client.doRequest(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating network", err.Error())
		return
	}

	diags := r.readIntoModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SetupKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SetupKeyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	reqURL := fmt.Sprintf("%s/api/networks/%s", r.client.BaseUrl, data.ID.ValueString())
	httpReq, err := http.NewRequest("DELETE", reqURL, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}

	_, err = r.client.doRequest(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting network", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *SetupKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
