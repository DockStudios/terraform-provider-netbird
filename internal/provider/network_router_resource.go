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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	netbirdApi "github.com/netbirdio/netbird/management/server/http/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &NetworkRouterResource{}
var _ resource.ResourceWithImportState = &NetworkRouterResource{}

func NewNetworkRouterResource() resource.Resource {
	return &NetworkRouterResource{}
}

// NetworkRouterResource defines the resource implementation.
type NetworkRouterResource struct {
	client *Client
}

// ExampleResourceModel describes the resource data model.
type NetworkRouterResourceModel struct {
	ID         types.String `tfsdk:"id"`
	NetworkId  types.String `tfsdk:"network_id"`
	Peer       types.String `tfsdk:"peer"`
	PeerGroups types.List   `tfsdk:"peer_groups"`
	Metric     types.Int32  `tfsdk:"metric"`
	Masquerade types.Bool   `tfsdk:"masquerade"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func (r *NetworkRouterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_router"
}

func (r *NetworkRouterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "NetworkRouter resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NetworkRouter ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "ID of the network to associate with",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"peer": schema.StringAttribute{
				MarkdownDescription: "Peer ID associated with route. This property can not be set together with peer_groups",
				Optional:            true,
			},
			"peer_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Peers Group IDs associated with route. This property can not be set together with peer",
				Optional:            true,
			},
			"metric": schema.Int32Attribute{
				MarkdownDescription: "Route metric number. Lowest number has higher priority",
				Optional:            true,
				Default:             int32default.StaticInt32(999),
				Computed:            true,
			},
			"masquerade": schema.BoolAttribute{
				MarkdownDescription: "Indicate if peer should masquerade traffic to this route's prefix",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Network router status",
				Required:            true,
			},
		},
	}
}

func (r *NetworkRouterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkRouterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkRouterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiData, diags := modelToApiRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if apiData == nil {
		resp.Diagnostics.AddError("nul pointer error", "Got nil pointer to NetworkRouterResourceModel")
		return
	}

	requestBody, err := json.Marshal(apiData)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	// Make API request
	reqURL := fmt.Sprintf("%s/api/networks/%s/routers", r.client.BaseUrl, data.NetworkId.ValueString())
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
	var responseData netbirdApi.NetworkRouter
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Assign values from API response
	data.ID = types.StringValue(responseData.Id)

	diags = r.readIntoModel(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkRouterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkRouterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.readIntoModel(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkRouterResource) readIntoModel(data *NetworkRouterResourceModel) diag.Diagnostics {
	// Update network model
	// Fetch data from API
	diags := diag.Diagnostics{}
	if data == nil {
		return diags
	}
	reqURL := fmt.Sprintf("%s/api/networks/%s/routers/%s", r.client.BaseUrl, data.NetworkId.ValueString(), data.ID.ValueString())
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
	// If not found
	if responseBody == nil {
		data.ID = types.StringNull()
		return diags
	}

	var responseData netbirdApi.NetworkRouter
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		diags.AddError("Error parsing response", err.Error())
		return diags
	}

	// Update state with latest data
	data.Peer = derefString(responseData.Peer)
	peerGroups, diags := convertStringSliceToListValue(derefStringSlice(responseData.PeerGroups))
	if diags.HasError() {
		return diags
	}
	data.PeerGroups = peerGroups

	data.Metric = types.Int32Value(int32(responseData.Metric))
	data.Enabled = types.BoolValue(responseData.Enabled)
	data.Masquerade = types.BoolValue(responseData.Masquerade)

	return diags
}

func modelToApiRequest(data NetworkRouterResourceModel) (*netbirdApi.NetworkRouterRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	peerGroups, diags := convertListToStringSlice(data.PeerGroups)
	if diags.HasError() {
		return nil, diags
	}

	return &netbirdApi.NetworkRouterRequest{
		Peer:       data.Peer.ValueStringPointer(),
		PeerGroups: &peerGroups,
		Metric:     int(data.Metric.ValueInt32()),
		Masquerade: data.Masquerade.ValueBool(),
		Enabled:    data.Enabled.ValueBool(),
	}, diags
}

func (r *NetworkRouterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NetworkRouterResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiData, diags := modelToApiRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if apiData == nil {
		resp.Diagnostics.AddError("nul pointer error", "Got nil pointer to NetworkRouterResourceModel")
		return
	}

	requestBody, err := json.Marshal(&apiData)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	reqURL := fmt.Sprintf("%s/api/networks/%s/routers/%s", r.client.BaseUrl, data.NetworkId.ValueString(), data.ID.ValueString())
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

	diags = r.readIntoModel(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NetworkRouterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkRouterResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	reqURL := fmt.Sprintf("%s/api/networks/%s/routers/%s", r.client.BaseUrl, data.NetworkId.ValueString(), data.ID.ValueString())
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

func (r *NetworkRouterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
