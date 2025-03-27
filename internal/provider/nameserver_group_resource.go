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
var _ resource.Resource = &NameserverGroupResource{}
var _ resource.ResourceWithImportState = &NameserverGroupResource{}

func NewNameserverGroupResource() resource.Resource {
	return &NameserverGroupResource{}
}

// NameserverGroupResource defines the resource implementation.
type NameserverGroupResource struct {
	client *Client
}

type NameserverResourceModel struct {
	Ip     types.String `tfsdk:"ip"`
	NsType types.String `tfsdk:"ns_type"`
	Port   types.Int32  `tfsdk:"port"`
}

// ExampleResourceModel describes the resource data model.
type NameserverGroupResourceModel struct {
	ID                   types.String              `tfsdk:"id"`
	Name                 types.String              `tfsdk:"name"`
	Description          types.String              `tfsdk:"description"`
	Nameservers          []NameserverResourceModel `tfsdk:"nameservers"`
	PeerGroups           types.List                `tfsdk:"peer_groups"`
	Domains              types.List                `tfsdk:"domains"`
	Primary              types.Bool                `tfsdk:"primary"`
	SearchDomainsEnabled types.Bool                `tfsdk:"search_domains_enabled"`
	Enabled              types.Bool                `tfsdk:"enabled"`
}

func (r *NameserverGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nameserver_group"
}

func (r *NameserverGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "NameserverGroup resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Nameserver Group ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Nameserver group name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the nameserver group",
				Optional:            true,
			},
			"peer_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Peer group IDs that defines group of peers that will use this nameserver group",
				Optional:            true,
			},
			"primary": schema.BoolAttribute{
				MarkdownDescription: "Defines if a nameserver group is primary that resolves all domains. It should be true only if domains list is empty.",
				Required:            true,
			},
			"nameservers": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Nameserver list",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							MarkdownDescription: "Nameserver IP",
							Required:            true,
						},
						"ns_type": schema.StringAttribute{
							MarkdownDescription: "Nameserver Type. E.g. `tcp` or `udp`",
							Required:            true,
						},
						"port": schema.Int32Attribute{
							MarkdownDescription: "Nameserver port",
							Required:            true,
						},
					},
				},
			},
			"domains": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Match domain list. It should be empty only if primary is true.",
				Required:            true,
			},
			"search_domains_enabled": schema.BoolAttribute{
				MarkdownDescription: "Search domain status for match domains. It should be true only if domains list is not empty.",
				Required:            true,
			},

			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Nameserver group status",
				Required:            true,
			},
		},
	}
}

func (r *NameserverGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NameserverGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data NameserverGroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiData, diags := nameserverGroupModelToApiRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if apiData == nil {
		resp.Diagnostics.AddError("nul pointer error", "Got nil pointer to NameserverGroupResourceModel")
		return
	}

	requestBody, err := json.Marshal(apiData)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	// Make API request
	reqURL := fmt.Sprintf("%s/api/dns/nameservers", r.client.BaseUrl)
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
	var responseData netbirdApi.NameserverGroup
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Assign values from API response
	data.ID = types.StringValue(responseData.Id)

	diags = r.readNameserverGroupIntoModel(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NameserverGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NameserverGroupResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.readNameserverGroupIntoModel(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NameserverGroupResource) readNameserverGroupIntoModel(data *NameserverGroupResourceModel) diag.Diagnostics {
	// Update network model
	// Fetch data from API
	diags := diag.Diagnostics{}
	if data == nil {
		return diags
	}
	reqURL := fmt.Sprintf("%s/api/dns/nameservers/%s", r.client.BaseUrl, data.ID.ValueString())
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

	var responseData netbirdApi.NameserverGroup
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		diags.AddError("Error parsing response", err.Error())
		return diags
	}

	data.Name = types.StringValue(responseData.Name)
	data.Description = nullStringToEmptyString(derefString(&responseData.Description))

	var nameservers []NameserverResourceModel
	for _, nameserver := range responseData.Nameservers {
		nameservers = append(nameservers, NameserverResourceModel{
			Ip:     types.StringValue(nameserver.Ip),
			NsType: types.StringValue(string(nameserver.NsType)),
			Port:   types.Int32Value(int32(nameserver.Port)),
		})
	}
	data.Nameservers = nameservers

	data.PeerGroups, diags = convertStringSliceToListValue(responseData.Groups)
	if diags.HasError() {
		return diags
	}

	data.Primary = types.BoolPointerValue(&responseData.Primary)

	data.Domains, diags = convertStringSliceToListValue(responseData.Domains)
	if diags.HasError() {
		return diags
	}

	data.SearchDomainsEnabled = types.BoolPointerValue(&responseData.SearchDomainsEnabled)
	data.Enabled = types.BoolPointerValue(&responseData.Enabled)

	return diags
}

func nameserverGroupModelToApiRequest(data NameserverGroupResourceModel) (*netbirdApi.NameserverGroupRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	peerGroups, diags := convertListToStringSlice(data.PeerGroups)
	if diags.HasError() {
		return nil, diags
	}

	domains, diags := convertListToStringSlice(data.Domains)
	if diags.HasError() {
		return nil, diags
	}

	var nameservers []netbirdApi.Nameserver
	for _, nameserverConfig := range data.Nameservers {
		nameservers = append(nameservers, netbirdApi.Nameserver{
			Ip:     nameserverConfig.Ip.ValueString(),
			NsType: netbirdApi.NameserverNsType(nameserverConfig.NsType.ValueString()),
			Port:   int(nameserverConfig.Port.ValueInt32()),
		})
	}

	return &netbirdApi.NameserverGroupRequest{
		Name:                 data.Name.ValueString(),
		Description:          data.Description.ValueString(),
		Nameservers:          nameservers,
		Groups:               peerGroups,
		Primary:              data.Primary.ValueBool(),
		Domains:              domains,
		SearchDomainsEnabled: data.SearchDomainsEnabled.ValueBool(),
		Enabled:              data.Enabled.ValueBool(),
	}, diags
}

func (r *NameserverGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NameserverGroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiData, diags := nameserverGroupModelToApiRequest(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if apiData == nil {
		resp.Diagnostics.AddError("nul pointer error", "Got nil pointer to NameserverGroupResourceModel")
		return
	}

	requestBody, err := json.Marshal(&apiData)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	reqURL := fmt.Sprintf("%s/api/dns/nameservers/%s", r.client.BaseUrl, data.ID.ValueString())
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

	diags = r.readNameserverGroupIntoModel(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NameserverGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NameserverGroupResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	reqURL := fmt.Sprintf("%s/api/dns/nameservers/%s", r.client.BaseUrl, data.ID.ValueString())
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

func (r *NameserverGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
