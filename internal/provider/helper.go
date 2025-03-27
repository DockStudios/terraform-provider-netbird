package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	netbirdApi "github.com/netbirdio/netbird/management/server/http/api"
)

// Helper function to convert PeerGroupBatch to PeerGroupDataSourceModel
func convertPeerGroups(groups []netbirdApi.GroupMinimum) []PeerGroupDataSourceModel {
	var convertedGroups []PeerGroupDataSourceModel
	for _, group := range groups {
		// Check if group.Issued is nil before dereferencing
		issued := ""
		if group.Issued != nil {
			issued = string(*group.Issued) // Safely dereference
		}
		convertedGroup := PeerGroupDataSourceModel{
			ID:             types.StringValue(group.Id),
			Name:           types.StringValue(group.Name),
			PeersCount:     types.Int64Value(int64(group.PeersCount)),
			ResourcesCount: types.Int64Value(int64(group.ResourcesCount)),
			Issued:         types.StringValue(issued),
		}
		convertedGroups = append(convertedGroups, convertedGroup)
	}
	return convertedGroups
}

// @TODO  Remove this
// Helper function to convert []string to []types.String
func convertStrings(input []string) []types.String {
	var output []types.String
	for _, str := range input {
		output = append(output, types.StringValue(str))
	}
	return output
}

func derefString(input *string) types.String {
	if input == nil {
		return types.StringNull()
	}
	return types.StringValue(*input)
}

func derefStringSlice(s *[]string) []string {
	if s == nil {
		return nil
	}
	return *s
}

// @TODO Delete this
func stringSliceToTerraform(apiValues []string) []types.String {
	var result []types.String
	for _, v := range apiValues {
		result = append(result, types.StringValue(v))
	}
	return result
}

func convertStringSliceToListValue(strings []string) (types.List, diag.Diagnostics) {
	var stringValueList []attr.Value
	var diags diag.Diagnostics
	for _, val := range strings {
		stringValueList = append(stringValueList, types.StringValue(val))
	}
	if len(stringValueList) == 0 {
		return types.ListNull(types.StringType), diags
	}

	listValue, diags := types.ListValue(types.StringType, stringValueList)
	if diags.HasError() {
		return types.ListNull(types.StringType), diags
	}
	return listValue, diags
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

func nullStringToEmptyString(input types.String) types.String {
	if input.ValueString() == "" {
		return types.StringNull()
	}
	return input
}
