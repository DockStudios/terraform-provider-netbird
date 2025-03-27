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
var _ resource.Resource = &NetworkResourceResource{}
var _ resource.ResourceWithImportState = &NetworkResourceResource{}

func NewNetworkResourceResource() resource.Resource {
	return &NetworkResourceResource{}
}

// NetworkResourceResource defines the resource implementation.
type NetworkResourceResource struct {
	client *Client
}

// ExampleResourceModel describes the resource data model.
type NetworkResourceResourceModel struct {
	ID          types.String `tfsdk:"id"`
	NetworkId   types.String `tfsdk:"network_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Address     types.String `tfsdk:"address"`
	PeerGroups  types.List   `tfsdk:"peer_groups"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

func (r *NetworkResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_resource"
}

func (r *NetworkResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "NetworkResource resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "NetworkResource ID",
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
			"name": schema.StringAttribute{
				MarkdownDescription: "Network resource name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Network resource description",
				Optional:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "Network resource address (either a direct host like 1.1.1.1 or 1.1.1.1/32, or a subnet like 192.168.178.0/24, or domains like example.com and *.example.com)",
				Required:            true,
			},
			"peer_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Group IDs containing the resource",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Network resource status",
				Required:            true,
			},
		},
	}
}

func (r *NetworkResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NetworkResourceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiData, diags := resourceModelToApiRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if apiData == nil {
		resp.Diagnostics.AddError("nul pointer error", "Got nil pointer to NetworkResourceResourceModel")
		return
	}

	requestBody, err := json.Marshal(apiData)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	// Make API request
	reqURL := fmt.Sprintf("%s/api/networks/%s/resources", r.client.BaseUrl, data.NetworkId.ValueString())
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
	var responseData netbirdApi.NetworkResource
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

func (r *NetworkResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NetworkResourceResourceModel

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

func (r *NetworkResourceResource) readIntoModel(data *NetworkResourceResourceModel) diag.Diagnostics {
	// Update network model
	// Fetch data from API
	diags := diag.Diagnostics{}
	if data == nil {
		return diags
	}
	reqURL := fmt.Sprintf("%s/api/networks/%s/resources/%s", r.client.BaseUrl, data.NetworkId.ValueString(), data.ID.ValueString())
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

	var responseData netbirdApi.NetworkResource
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		diags.AddError("Error parsing response", err.Error())
		return diags
	}

	// Update state with latest data
	data.Name = types.StringValue(responseData.Name)
	data.Description = derefString(responseData.Description)
	peerGroups, diags := convertGroupMinimumToIdList(&responseData.Groups)
	if diags.HasError() {
		return diags
	}
	data.PeerGroups = peerGroups

	data.Address = types.StringValue(responseData.Address)
	data.Enabled = types.BoolValue(responseData.Enabled)

	return diags
}

func resourceModelToApiRequest(data NetworkResourceResourceModel) (*netbirdApi.NetworkResourceRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	peerGroups, diags := convertListToStringSlice(data.PeerGroups)
	if diags.HasError() {
		return nil, diags
	}

	return &netbirdApi.NetworkResourceRequest{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		Groups:      peerGroups,
		Address:     data.Address.ValueString(),
		Enabled:     data.Enabled.ValueBool(),
	}, diags
}

func (r *NetworkResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NetworkResourceResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiData, diags := resourceModelToApiRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if apiData == nil {
		resp.Diagnostics.AddError("nul pointer error", "Got nil pointer to NetworkResourceResourceModel")
		return
	}

	requestBody, err := json.Marshal(&apiData)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	reqURL := fmt.Sprintf("%s/api/networks/%s/resources/%s", r.client.BaseUrl, data.NetworkId.ValueString(), data.ID.ValueString())
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

func (r *NetworkResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NetworkResourceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	reqURL := fmt.Sprintf("%s/api/networks/%s/resources/%s", r.client.BaseUrl, data.NetworkId.ValueString(), data.ID.ValueString())
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

func (r *NetworkResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
