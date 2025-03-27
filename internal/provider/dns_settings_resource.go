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
var _ resource.Resource = &DnsSettingsResource{}
var _ resource.ResourceWithImportState = &DnsSettingsResource{}

func NewDnsSettingsResource() resource.Resource {
	return &DnsSettingsResource{}
}

// DnsSettingsResource defines the resource implementation.
type DnsSettingsResource struct {
	client *Client
}

type DnsSettingsResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	DisabledManagementGroups types.List   `tfsdk:"disabled_management_groups"`
}

func (r *DnsSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_settings"
}

func (r *DnsSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "DNS Settings resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Config ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disabled_management_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Groups whose DNS management is disabled",
				Required:            true,
			},
		},
	}
}

func (r *DnsSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func dnsSettingsModelToApi(data *DnsSettingsResourceModel) (netbirdApi.DNSSettings, diag.Diagnostics) {
	var diags diag.Diagnostics
	var apiModel netbirdApi.DNSSettings
	apiModel.DisabledManagementGroups, diags = convertListToStringSlice(data.DisabledManagementGroups)
	return apiModel, diags
}

func (r *DnsSettingsResource) updateDnsSettings(data *DnsSettingsResourceModel) ([]byte, diag.Diagnostics) {
	apiModel, diags := dnsSettingsModelToApi(data)
	if diags.HasError() {
		return nil, diags
	}

	requestBody, err := json.Marshal(apiModel)
	if err != nil {
		diags.AddError("Error marshaling request body", err.Error())
		return nil, diags
	}

	// Make API request
	reqURL := fmt.Sprintf("%s/api/dns/settings", r.client.BaseUrl)
	httpReq, err := http.NewRequest("PUT", reqURL, bytes.NewBuffer(requestBody))
	if err != nil {
		diags.AddError("Error creating request", err.Error())
		return nil, diags
	}
	httpReq.Header.Set("Content-Type", "application/json")

	responseBody, err := r.client.doRequest(httpReq)
	if err != nil {
		diags.AddError("Error making API request", err.Error())
		return nil, diags
	}
	return responseBody, diags
}

func (r *DnsSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DnsSettingsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	responseBody, diags := r.updateDnsSettings(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse response
	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	// Assign hard coded value
	data.ID = types.StringValue("dns-settings")

	diags = r.readDnsSettingsIntoModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DnsSettingsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.readDnsSettingsIntoModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsSettingsResource) readDnsSettingsIntoModel(ctx context.Context, data *DnsSettingsResourceModel) diag.Diagnostics {
	// Update network model
	// Fetch data from API
	diags := diag.Diagnostics{}
	reqURL := fmt.Sprintf("%s/api/dns/settings", r.client.BaseUrl)
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

	var responseData netbirdApi.DNSSettings
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		diags.AddError("Error parsing response", err.Error())
		return diags
	}

	disabledManagementGroups, newDiags := types.ListValueFrom(ctx, types.StringType, responseData.DisabledManagementGroups)
	diags.Append(newDiags...)
	data.DisabledManagementGroups = disabledManagementGroups

	return diags
}

func (r *DnsSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DnsSettingsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, diags := r.updateDnsSettings(&data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.readDnsSettingsIntoModel(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DnsSettingsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	requestBody, err := json.Marshal(netbirdApi.DNSSettings{
		DisabledManagementGroups: []string{},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling request body", err.Error())
		return
	}

	reqURL := fmt.Sprintf("%s/api/dns/settings", r.client.BaseUrl)
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

	resp.State.RemoveResource(ctx)
}

func (r *DnsSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
