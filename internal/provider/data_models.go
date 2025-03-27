package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type PeerDataSourceModel struct {
	ID                          types.String               `tfsdk:"id"`
	Name                        types.String               `tfsdk:"name"`
	IP                          types.String               `tfsdk:"ip"`
	ConnectionIP                types.String               `tfsdk:"connection_ip"`
	Connected                   types.Bool                 `tfsdk:"connected"`
	LastSeen                    types.String               `tfsdk:"last_seen"`
	OS                          types.String               `tfsdk:"os"`
	KernelVersion               types.String               `tfsdk:"kernel_version"`
	GeonameID                   types.Int64                `tfsdk:"geoname_id"`
	Version                     types.String               `tfsdk:"version"`
	Groups                      []PeerGroupDataSourceModel `tfsdk:"groups"`
	SSHEnabled                  types.Bool                 `tfsdk:"ssh_enabled"`
	UserID                      types.String               `tfsdk:"user_id"`
	Hostname                    types.String               `tfsdk:"hostname"`
	UIVersion                   types.String               `tfsdk:"ui_version"`
	DNSLabel                    types.String               `tfsdk:"dns_label"`
	LoginExpirationEnabled      types.Bool                 `tfsdk:"login_expiration_enabled"`
	LoginExpired                types.Bool                 `tfsdk:"login_expired"`
	LastLogin                   types.String               `tfsdk:"last_login"`
	InactivityExpirationEnabled types.Bool                 `tfsdk:"inactivity_expiration_enabled"`
	ApprovalRequired            types.Bool                 `tfsdk:"approval_required"`
	CountryCode                 types.String               `tfsdk:"country_code"`
	CityName                    types.String               `tfsdk:"city_name"`
	SerialNumber                types.String               `tfsdk:"serial_number"`
	ExtraDNSLabels              []types.String             `tfsdk:"extra_dns_labels"`
	AccessiblePeersCount        types.Int64                `tfsdk:"accessible_peers_count"`
}

type PeerGroupDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	PeersCount     types.Int64  `tfsdk:"peers_count"`
	ResourcesCount types.Int64  `tfsdk:"resources_count"`
	Issued         types.String `tfsdk:"issued"`
}

type PeersDataSourceModel struct {
	Name  types.String          `tfsdk:"name"`
	IP    types.String          `tfsdk:"ip"`
	Peers []PeerDataSourceModel `tfsdk:"peers"`
}
