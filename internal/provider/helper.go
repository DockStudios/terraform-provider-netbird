package provider

import (
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

// Helper function to convert []string to []types.String
func convertStrings(input []string) []types.String {
	var output []types.String
	for _, str := range input {
		output = append(output, types.StringValue(str))
	}
	return output
}
