package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	netbirdApi "github.com/netbirdio/netbird/management/server/http/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithImportState = &PolicyResource{}

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

// PolicyResource defines the resource implementation.
type PolicyResource struct {
	client *Client
}

type PolicyModel struct {
	ID                  types.String      `tfsdk:"id"`
	Name                types.String      `tfsdk:"name"`
	Description         types.String      `tfsdk:"description"`
	Enabled             types.Bool        `tfsdk:"enabled"`
	SourcePostureChecks types.List        `tfsdk:"source_posture_checks"`
	Rules               []PolicyRuleModel `tfsdk:"rules"`
}

// ResourceModel represents a source or destination resource in a policy.
type ResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

// PolicyRuleModel represents an individual rule within a policy.
type PolicyRuleModel struct {
	ID                  types.String     `tfsdk:"id"`
	Name                types.String     `tfsdk:"name"`
	Description         types.String     `tfsdk:"description"`
	Enabled             types.Bool       `tfsdk:"enabled"`
	Action              types.String     `tfsdk:"action"`
	Bidirectional       types.Bool       `tfsdk:"bidirectional"`
	Protocol            types.String     `tfsdk:"protocol"`
	Ports               types.List       `tfsdk:"ports"`
	PortRanges          []PortRangeModel `tfsdk:"port_ranges"`
	Sources             types.List       `tfsdk:"sources"`
	Destinations        types.List       `tfsdk:"destinations"`
	SourceResource      *ResourceModel   `tfsdk:"source_resource"`
	DestinationResource *ResourceModel   `tfsdk:"destination_resource"`
}

// PortRangeModel represents a range of ports in a policy rule.
type PortRangeModel struct {
	Start types.Int32 `tfsdk:"start"`
	End   types.Int32 `tfsdk:"end"`
}

func (r *PolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Policy resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Policy ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Policy Name",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Policy description",
				Default:     stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Policy status",
			},
			"source_posture_checks": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of source posture check IDs",
				Optional:            true,
				Computed:            true,
			},
			"rules": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "List of policy rules",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Rule ID",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Rule name",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule description",
							Default:     stringdefault.StaticString(""),
						},
						"enabled": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Rule status",
						},
						"action": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Policy rule `accept` or `drop` packets",
						},
						"bidirectional": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Define if the rule is applicable in both directions, sources, and destinations",
						},
						"protocol": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Traffic protocol, e.g. `tcp`, `udp`, `icmp`",
						},
						"ports": schema.ListAttribute{
							ElementType:         types.StringType,
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "List of affected ports",
						},
						"port_ranges": schema.ListNestedAttribute{
							Optional:            true,
							MarkdownDescription: "List of port ranges affecting policy rule",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"start": schema.Int32Attribute{
										Required:            true,
										MarkdownDescription: "Start port",
									},
									"end": schema.Int32Attribute{
										Required:            true,
										MarkdownDescription: "End port",
									},
								},
							},
						},
						"sources": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "Policy rule source group IDs",
							Optional:            true,
						},
						"destinations": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "Policy rule destination group IDs",
							Optional:            true,
						},
						"source_resource": schema.SingleNestedAttribute{
							Optional:            true,
							MarkdownDescription: "Source resources",
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "ID of the resource",
								},
								"type": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Network resource type based of the address",
								},
							},
						},
						"destination_resource": schema.ListNestedAttribute{
							Optional:            true,
							MarkdownDescription: "Source resources",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "ID of the resource",
									},
									"type": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "Network resource type based of the address",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *PolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func convertToRulesResourcesApiModel(modelResource *ResourceModel) (*netbirdApi.Resource, diag.Diagnostics) {
	var diags diag.Diagnostics
	if modelResource == nil {
		return nil, diags
	}
	return &netbirdApi.Resource{
		Id:   modelResource.ID.ValueString(),
		Type: netbirdApi.ResourceType(modelResource.Type.ValueString()),
	}, diags
}

func convertToRulesPortRangesApiModel(modelRanges *[]PortRangeModel) ([]netbirdApi.RulePortRange, diag.Diagnostics) {
	var portRanges []netbirdApi.RulePortRange
	var diags diag.Diagnostics

	if modelRanges == nil {
		return portRanges, diags
	}

	for _, portRange := range *modelRanges {
		portRanges = append(portRanges, netbirdApi.RulePortRange{
			Start: int(portRange.Start.ValueInt32()),
			End:   int(portRange.End.ValueInt32()),
		})
	}

	return portRanges, diags
}

func convertToRulesUpdateApiModel(modelRules *[]PolicyRuleModel) ([]netbirdApi.PolicyRuleUpdate, diag.Diagnostics) {
	var apiRules []netbirdApi.PolicyRuleUpdate
	if modelRules == nil {
		return apiRules, nil
	}
	var diags diag.Diagnostics
	for _, modelRule := range *modelRules {

		ports, newDiags := convertListToStringSlice(modelRule.Ports)
		diags.Append(newDiags...)
		if diags.HasError() {
			return apiRules, diags
		}

		portRanges, newDiags := convertToRulesPortRangesApiModel(&modelRule.PortRanges)
		diags.Append(newDiags...)
		if diags.HasError() {
			return apiRules, diags
		}

		sources, newDiags := convertListToStringSlice(modelRule.Sources)
		diags.Append(newDiags...)
		if diags.HasError() {
			return apiRules, diags
		}

		sourceResource, diags := convertToRulesResourcesApiModel(modelRule.SourceResource)
		if diags.HasError() {
			return apiRules, diags
		}

		destinations, diags := convertListToStringSlice(modelRule.Destinations)
		if diags.HasError() {
			return apiRules, diags
		}

		destinationResource, diags := convertToRulesResourcesApiModel(modelRule.SourceResource)
		if diags.HasError() {
			return apiRules, diags
		}

		apiRules = append(apiRules, netbirdApi.PolicyRuleUpdate{
			Name:                modelRule.Name.ValueString(),
			Description:         modelRule.Description.ValueStringPointer(),
			Enabled:             modelRule.Enabled.ValueBool(),
			Action:              netbirdApi.PolicyRuleUpdateAction(modelRule.Action.ValueString()),
			Bidirectional:       modelRule.Bidirectional.ValueBool(),
			Protocol:            netbirdApi.PolicyRuleUpdateProtocol(modelRule.Protocol.ValueString()),
			Ports:               &ports,
			PortRanges:          &portRanges,
			Sources:             &sources,
			SourceResource:      sourceResource,
			Destinations:        &destinations,
			DestinationResource: destinationResource,
		})
		return apiRules, diags
	}

	return apiRules, diags
}

func convertResourceModel(resource *netbirdApi.Resource) *ResourceModel {
	if resource == nil {
		return nil
	}
	return &ResourceModel{
		ID:   types.StringValue(resource.Id),
		Type: types.StringValue(string(resource.Type)),
	}
}

func convertPortRangesToList(portRanges *[]netbirdApi.RulePortRange) []PortRangeModel {
	var terraformPortRanges []PortRangeModel
	if portRanges == nil {
		return nil
	}

	for _, portRange := range *portRanges {
		terraformPortRanges = append(terraformPortRanges, PortRangeModel{
			Start: types.Int32Value(int32(portRange.Start)),
			End:   types.Int32Value(int32(portRange.End)),
		})
	}
	return terraformPortRanges
}

func convertGroupMinimumToIdList(groupList *[]netbirdApi.GroupMinimum) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	var idList []string
	if groupList == nil {
		return types.ListNull(types.StringType), diags
	}

	for _, group := range *groupList {
		idList = append(idList, group.Id)
	}

	return convertStringSliceToListValue(idList)
}

func convertRulesFromAPI(data *[]netbirdApi.PolicyRule) ([]PolicyRuleModel, diag.Diagnostics) {
	var rules []PolicyRuleModel
	var diags diag.Diagnostics

	if data == nil {
		return rules, diags
	}

	for _, dataRule := range *data {

		ports, diags := convertStringSliceToListValue(derefStringSlice(dataRule.Ports))
		if diags.HasError() {
			return rules, diags
		}

		sources, diags := convertGroupMinimumToIdList(dataRule.Sources)
		if diags.HasError() {
			return rules, diags
		}

		destinations, diags := convertGroupMinimumToIdList(dataRule.Destinations)
		if diags.HasError() {
			return rules, diags
		}

		rules = append(rules, PolicyRuleModel{
			ID:                  derefString(dataRule.Id),
			Name:                types.StringValue(dataRule.Name),
			Description:         derefString(dataRule.Description),
			Enabled:             types.BoolValue(dataRule.Enabled),
			Action:              types.StringValue(string(dataRule.Action)), // Assuming Action is an enum and needs to be converted
			Bidirectional:       types.BoolValue(dataRule.Bidirectional),
			Protocol:            types.StringValue(string(dataRule.Protocol)), // Assuming Protocol is a string or enum
			Ports:               ports,
			PortRanges:          convertPortRangesToList(dataRule.PortRanges),
			Sources:             sources,
			Destinations:        destinations,
			SourceResource:      convertResourceModel(dataRule.SourceResource),
			DestinationResource: convertResourceModel(dataRule.DestinationResource),
		})
	}

	return rules, diags
}

func convertPolicyFromApiModel(data netbirdApi.Policy) (PolicyModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var policyModel PolicyModel

	policyModel.ID = derefString(data.Id)
	policyModel.Name = types.StringValue(data.Name)
	policyModel.Description = derefString(data.Description)
	policyModel.Enabled = types.BoolValue(data.Enabled)

	var sourcePostureChecks []attr.Value
	for _, val := range data.SourcePostureChecks {
		sourcePostureChecks = append(sourcePostureChecks, types.StringValue(val))
	}
	sourcePostureChecksListValue, diags := types.ListValue(types.StringType, sourcePostureChecks)
	if diags.HasError() {
		return policyModel, diags
	}
	policyModel.SourcePostureChecks = sourcePostureChecksListValue

	rules, diags := convertRulesFromAPI(&data.Rules)
	if diags.HasError() {
		return policyModel, diags
	}
	policyModel.Rules = rules

	return policyModel, diags
}

func convertListToStringSlice(list basetypes.ListValue) ([]string, diag.Diagnostics) {
	result := []string{}
	var diags diag.Diagnostics

	// Handle null or unknown values
	if list.IsNull() || list.IsUnknown() {
		return result, nil
	}

	// Extract elements
	elements := list.Elements() // Get list of attr.Value
	for _, elem := range elements {
		strVal, ok := elem.(basetypes.StringValue)
		if !ok {
			diags.AddError("Unexpected type", fmt.Sprintf("unexpected type: %T", elem))
			return nil, diags
		}
		result = append(result, strVal.ValueString()) // Convert to native Go string
	}

	return result, nil
}

func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform list of peers to a Go slice
	sourcePostureChecks, diags := convertListToStringSlice(data.SourcePostureChecks)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, diags := convertToRulesUpdateApiModel(&data.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy := netbirdApi.PolicyCreate{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueStringPointer(),
		Enabled:             data.Enabled.ValueBool(),
		SourcePostureChecks: &sourcePostureChecks,
		Rules:               rules,
	}
	jsonData, err := json.Marshal(policy)
	if err != nil {
		resp.Diagnostics.AddError("JSON Encoding Error", err.Error())
		return
	}

	tflog.Info(ctx, string(jsonData[:]))
	request, err := http.NewRequest("POST", r.client.BaseUrl+"/api/policies", bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Diagnostics.AddError("Request Creation Error", err.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	body, err := r.client.doRequest(request)
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	var createdPolicy netbirdApi.Policy
	if err := json.Unmarshal(body, &createdPolicy); err != nil {
		resp.Diagnostics.AddError("JSON Decoding Error", err.Error())
		return
	}

	data, diags = convertPolicyFromApiModel(createdPolicy)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch data from API
	reqURL := fmt.Sprintf("%s/api/policies/%s", r.client.BaseUrl, data.ID.ValueString())
	httpReq, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", err.Error())
		return
	}

	responseBody, err := r.client.doRequest(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Error fetching policy", err.Error())
		return
	}

	// Handle when resource does not exist
	if responseBody == nil {
		data.ID = types.StringNull()
		return
	}

	var responseData netbirdApi.Policy
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		resp.Diagnostics.AddError("Error parsing response", err.Error())
		return
	}

	data, diags := convertPolicyFromApiModel(responseData)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PolicyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform list of peers to a Go slice
	sourcePostureChecks, diags := convertListToStringSlice(data.SourcePostureChecks)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, diags := convertToRulesUpdateApiModel(&data.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy := netbirdApi.PolicyUpdate{
		Name:                data.Name.ValueString(),
		Description:         data.Description.ValueStringPointer(),
		Enabled:             data.Enabled.ValueBool(),
		SourcePostureChecks: &sourcePostureChecks,
		Rules:               rules,
	}
	jsonData, err := json.Marshal(policy)
	if err != nil {
		resp.Diagnostics.AddError("JSON Encoding Error", err.Error())
		return
	}

	url := fmt.Sprintf("%s/api/policies/%s", r.client.BaseUrl, data.ID.ValueString())
	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		resp.Diagnostics.AddError("Request Creation Error", err.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	body, err := r.client.doRequest(request)
	if err != nil {
		resp.Diagnostics.AddError("API Error", err.Error())
		return
	}

	var createdPolicy netbirdApi.Policy
	if err := json.Unmarshal(body, &createdPolicy); err != nil {
		resp.Diagnostics.AddError("JSON Decoding Error", err.Error())
		return
	}

	data, diags = convertPolicyFromApiModel(createdPolicy)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	reqURL := fmt.Sprintf("%s/api/policies/%s", r.client.BaseUrl, data.ID.ValueString())
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

func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
