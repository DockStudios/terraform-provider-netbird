package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type PeerDataSourceModel struct {
	ID                          types.String               `tfsdk:"id" json:"id"`
	Name                        types.String               `tfsdk:"name" json:"name"`
	IP                          types.String               `tfsdk:"ip" json:"ip"`
	ConnectionIP                types.String               `tfsdk:"connection_ip" json:"connection_ip"`
	Connected                   types.Bool                 `tfsdk:"connected" json:"connected"`
	LastSeen                    types.String               `tfsdk:"last_seen" json:"last_seen"`
	OS                          types.String               `tfsdk:"os" json:"os"`
	KernelVersion               types.String               `tfsdk:"kernel_version" json:"kernel_version"`
	GeonameID                   types.Int64                `tfsdk:"geoname_id" json:"geoname_id"`
	Version                     types.String               `tfsdk:"version" json:"version"`
	Groups                      []PeerGroupDataSourceModel `tfsdk:"groups" json:"groups"`
	SSHEnabled                  types.Bool                 `tfsdk:"ssh_enabled" json:"ssh_enabled"`
	UserID                      types.String               `tfsdk:"user_id" json:"user_id"`
	Hostname                    types.String               `tfsdk:"hostname" json:"hostname"`
	UIVersion                   types.String               `tfsdk:"ui_version" json:"ui_version"`
	DNSLabel                    types.String               `tfsdk:"dns_label" json:"dns_label"`
	LoginExpirationEnabled      types.Bool                 `tfsdk:"login_expiration_enabled" json:"login_expiration_enabled"`
	LoginExpired                types.Bool                 `tfsdk:"login_expired" json:"login_expired"`
	LastLogin                   types.String               `tfsdk:"last_login" json:"last_login"`
	InactivityExpirationEnabled types.Bool                 `tfsdk:"inactivity_expiration_enabled" json:"inactivity_expiration_enabled"`
	ApprovalRequired            types.Bool                 `tfsdk:"approval_required" json:"approval_required"`
	CountryCode                 types.String               `tfsdk:"country_code" json:"country_code"`
	CityName                    types.String               `tfsdk:"city_name" json:"city_name"`
	SerialNumber                types.String               `tfsdk:"serial_number" json:"serial_number"`
	ExtraDNSLabels              []types.String             `tfsdk:"extra_dns_labels" json:"extra_dns_labels"`
	AccessiblePeersCount        types.Int64                `tfsdk:"accessible_peers_count" json:"accessible_peers_count"`
}

type PeerGroupDataSourceModel struct {
	ID             types.String `tfsdk:"id" json:"id"`
	Name           types.String `tfsdk:"name" json:"name"`
	PeersCount     types.Int64  `tfsdk:"peers_count" json:"peers_count"`
	ResourcesCount types.Int64  `tfsdk:"resources_count" json:"resources_count"`
	Issued         types.String `tfsdk:"issued" json:"issued"`
}

type PeersDataSourceModel struct {
	Name  types.String          `tfsdk:"name"`
	IP    types.String          `tfsdk:"ip"`
	Peers []PeerDataSourceModel `tfsdk:"peers"`
}
