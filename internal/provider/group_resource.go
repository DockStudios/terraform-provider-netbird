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
var _ resource.Resource = &GroupResource{}
var _ resource.ResourceWithImportState = &GroupResource{}

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

// GroupResource defines the resource implementation.
type GroupResource struct {
	client *Client
}

// Group resource (resource) model
type GroupResourceResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

// GroupResourceModel describes the resource data model.
type GroupResourceModel struct {
	ID             types.String                 `tfsdk:"id"`
	Name           types.String                 `tfsdk:"name"`
	Peers          types.List                   `tfsdk:"peers"`
	Resources      []GroupResourceResourceModel `tfsdk:"resources"`
	PeersCount     types.Int64                  `tfsdk:"peers_count"`
	ResourcesCount types.Int64                  `tfsdk:"resources_count"`
	Issued         types.String                 `tfsdk:"issued"`
}

func (r *GroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Group resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Group ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Group Name",
				Required:            true,
			},
			"peers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of associated peers IDs",
				Optional:            true,
			},
			"resources": schema.ListNestedAttribute{
				Optional:    true,
				Description: "List of groups associated with the peer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "Unique identifier of the resource.",
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Type of the resource. Must of one of: `host`.",
						},
					},
				},
			},
			"peers_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Count of peers associated with the group.",
			},
			"resources_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Count of resources associated with the group.",
			},
			"issued": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "How the group was issued (e.g., `api`, `integration`, `jwt`).",
			},
		},
	}
}

func (r *GroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform list of peers to a Go slice
	var peersList []string
	resp.Diagnostics.Append(data.Peers.ElementsAs(ctx, &peersList, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform list of resources to Go slice
	var resourcesList []netbirdApi.Resource
	for _, res := range data.Resources {
		resourcesList = append(resourcesList, netbirdApi.Resource{
			Id:   res.ID.ValueString(),
			Type: netbirdApi.ResourceType(res.Type.ValueString()),
		})
	}

	// Prepare request body
	requestBody, err := json.Marshal(netbirdApi.GroupRequest{
		Name:      data.Name.ValueString(),
		Peers:     &peersList,
		Resources: &resourcesList,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	// API request
	reqURL := fmt.Sprintf("%s/api/groups", r.client.BaseUrl)
	httpReq, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(requestBody))
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	responseBody, err := r.client.doRequest(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating group", err.Error())
		return
	}

	// Parse response
	var responseData netbirdApi.Group
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Set state values
	data.ID = types.StringValue(responseData.Id)
	data.PeersCount = types.Int64Value(int64(responseData.PeersCount))
	data.ResourcesCount = types.Int64Value(int64(responseData.ResourcesCount))
	if responseData.Issued != nil {
		data.Issued = types.StringValue(string(*responseData.Issued))
	}

	// Update state with response data
	var updatedPeersList []string
	for _, peer := range responseData.Peers {
		updatedPeersList = append(updatedPeersList, peer.Id)
	}
	var diags diag.Diagnostics
	data.Peers, diags = types.ListValueFrom(ctx, types.StringType, updatedPeersList)
	resp.Diagnostics.Append(diags...)

	var updatedResourcesList []GroupResourceResourceModel
	for _, res := range responseData.Resources {
		updatedResourcesList = append(updatedResourcesList, GroupResourceResourceModel{
			ID:   types.StringValue(res.Id),
			Type: types.StringValue(string(res.Type)),
		})
	}
	data.Resources = updatedResourcesList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch data from API
	reqURL := fmt.Sprintf("%s/api/groups/%s", r.client.BaseUrl, data.ID.ValueString())
	httpReq, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}

	responseBody, err := r.client.doRequest(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching group", err.Error())
		return
	}

	// Handle when resource does not exist
	if responseBody == nil {
		data.ID = types.StringNull()
		return
	}

	var responseData netbirdApi.Group
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Update state with latest data
	data.Name = types.StringValue(responseData.Name)
	data.PeersCount = types.Int64Value(int64(responseData.PeersCount))
	data.ResourcesCount = types.Int64Value(int64(responseData.ResourcesCount))
	if responseData.Issued != nil {
		data.Issued = types.StringValue(string(*responseData.Issued))
	}

	// Convert peers list
	var peersList []string
	for _, peer := range responseData.Peers {
		peersList = append(peersList, peer.Id)
	}
	var diags diag.Diagnostics
	data.Peers, diags = types.ListValueFrom(ctx, types.StringType, peersList)
	resp.Diagnostics.Append(diags...)

	// Convert resources list
	var resourcesList []GroupResourceResourceModel
	for _, res := range responseData.Resources {
		resourcesList = append(resourcesList, GroupResourceResourceModel{
			ID:   types.StringValue(res.Id),
			Type: types.StringValue(string(res.Type)),
		})
	}
	data.Resources = resourcesList

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform list of peers to a Go slice
	var peersList []string
	resp.Diagnostics.Append(data.Peers.ElementsAs(ctx, &peersList, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform list of resources to Go slice
	var resourcesList []netbirdApi.Resource
	for _, res := range data.Resources {
		resourcesList = append(resourcesList, netbirdApi.Resource{
			Id:   res.ID.ValueString(),
			Type: netbirdApi.ResourceType(res.Type.ValueString()),
		})
	}

	// Prepare request body
	requestBody, err := json.Marshal(netbirdApi.GroupRequest{
		Name:      data.Name.ValueString(),
		Peers:     &peersList,
		Resources: &resourcesList,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	// API request
	reqURL := fmt.Sprintf("%s/api/groups/%s", r.client.BaseUrl, data.ID.ValueString())
	httpReq, err := http.NewRequest("PUT", reqURL, bytes.NewBuffer(requestBody))
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	responseBody, err := r.client.doRequest(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating group", err.Error())
		return
	}

	// Parse response
	var responseData netbirdApi.Group
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Set state values
	data.ID = types.StringValue(responseData.Id)
	data.PeersCount = types.Int64Value(int64(responseData.PeersCount))
	data.ResourcesCount = types.Int64Value(int64(responseData.ResourcesCount))
	if responseData.Issued != nil {
		data.Issued = types.StringValue(string(*responseData.Issued))
	}

	// Update state with response data
	var updatedPeersList []string
	for _, peer := range responseData.Peers {
		updatedPeersList = append(updatedPeersList, peer.Id)
	}
	var diags diag.Diagnostics
	data.Peers, diags = types.ListValueFrom(ctx, types.StringType, updatedPeersList)
	resp.Diagnostics.Append(diags...)

	var updatedResourcesList []GroupResourceResourceModel
	for _, res := range responseData.Resources {
		updatedResourcesList = append(updatedResourcesList, GroupResourceResourceModel{
			ID:   types.StringValue(res.Id),
			Type: types.StringValue(string(res.Type)),
		})
	}
	data.Resources = updatedResourcesList

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	reqURL := fmt.Sprintf("%s/api/groups/%s", r.client.BaseUrl, data.ID.ValueString())
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

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
