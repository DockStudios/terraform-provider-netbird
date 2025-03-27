package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	netbirdApi "github.com/netbirdio/netbird/management/server/http/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PeerDataSource{}

func NewPeerDataSource() datasource.DataSource {
	return &PeerDataSource{}
}

// PeerDataSource defines the data source implementation.
type PeerDataSource struct {
	client *Client
}

func (d *PeerDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peer"
}

func (d *PeerDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Retrieve peer details",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Unique identifier of the peer.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the peer.",
			},
			"ip": schema.StringAttribute{
				Computed:    true,
				Description: "IP address of the peer.",
			},
			"connection_ip": schema.StringAttribute{
				Computed:    true,
				Description: "IP address used for connections to the peer.",
			},
			"connected": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates whether the peer is currently connected.",
			},
			"last_seen": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last time the peer was seen.",
			},
			"os": schema.StringAttribute{
				Computed:    true,
				Description: "Operating system running on the peer.",
			},
			"kernel_version": schema.StringAttribute{
				Computed:    true,
				Description: "Kernel version of the peer's operating system.",
			},
			"geoname_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Geoname identifier for the peer's location.",
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "Version of the peer software.",
			},
			"groups": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of groups associated with the peer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Unique identifier of the group.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Name of the group.",
						},
						"peers_count": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of Peers in the group.",
						},
						"resources_count": schema.Int64Attribute{
							Computed:    true,
							Description: "Number of resources in the group.",
						},
						"issued": schema.StringAttribute{
							Computed:    true,
							Description: "Timestamp when the group was issued.",
						},
					},
				},
			},
			"ssh_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates whether SSH access is enabled for the peer.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "User identifier associated with the peer.",
			},
			"hostname": schema.StringAttribute{
				Computed:    true,
				Description: "Hostname of the peer.",
			},
			"ui_version": schema.StringAttribute{
				Computed:    true,
				Description: "Version of the UI associated with the peer.",
			},
			"dns_label": schema.StringAttribute{
				Computed:    true,
				Description: "DNS label assigned to the peer.",
			},
			"login_expiration_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates whether login expiration is enabled for the peer.",
			},
			"login_expired": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates whether the peer's login has expired.",
			},
			"last_login": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last user login to the peer.",
			},
			"inactivity_expiration_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates whether inactivity-based expiration is enabled for the peer.",
			},
			"approval_required": schema.BoolAttribute{
				Computed:    true,
				Description: "Indicates whether approval is required for the peer to access resources.",
			},
			"country_code": schema.StringAttribute{
				Computed:    true,
				Description: "ISO country code of the peer's location.",
			},
			"city_name": schema.StringAttribute{
				Computed:    true,
				Description: "City name of the peer's location.",
			},
			"serial_number": schema.StringAttribute{
				Computed:    true,
				Description: "Serial number of the peer.",
			},
			"extra_dns_labels": schema.ListAttribute{
				Computed:    true,
				Description: "Additional DNS labels assigned to the peer.",
				ElementType: types.StringType,
			},
			"accessible_peers_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of Peers accessible by this peer.",
			},
		},
	}
}

func (d *PeerDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *PeerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PeerDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.ID.String() == "" {
		resp.Diagnostics.AddAttributeError(path.Root("id"), "ID is invalid", "ID must be set to a valid string")
	}

	tflog.Info(ctx, "ID: "+data.ID.String())
	endpoint := fmt.Sprintf("%s/api/peers/%s", d.client.BaseUrl, data.ID.ValueString())

	reqHTTP, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Request", err.Error())
		return
	}

	body, err := d.client.doRequest(reqHTTP)
	if err != nil {
		resp.Diagnostics.AddError("Error Making API Request: "+endpoint, err.Error())
		return
	}

	tflog.Info(ctx, "Obtained Peer data source response: "+string(body[:]))
	var peerBatch netbirdApi.PeerBatch
	if err := json.Unmarshal(body, &peerBatch); err != nil {
		resp.Diagnostics.AddError("Error Parsing API Response", err.Error())
		return
	}

	data.ID = types.StringValue(peerBatch.Id)
	data.Name = types.StringValue(peerBatch.Name)
	data.IP = types.StringValue(peerBatch.Ip)
	data.ConnectionIP = types.StringValue(peerBatch.ConnectionIp)
	data.Connected = types.BoolValue(peerBatch.Connected)
	data.LastSeen = types.StringValue(peerBatch.LastSeen.String())
	data.OS = types.StringValue(peerBatch.Os)
	data.KernelVersion = types.StringValue(peerBatch.KernelVersion)
	data.GeonameID = types.Int64Value(int64(peerBatch.GeonameId))
	data.Version = types.StringValue(peerBatch.Version)
	data.Groups = convertPeerGroups(peerBatch.Groups) // Helper function to convert groups
	data.SSHEnabled = types.BoolValue(peerBatch.SshEnabled)
	data.UserID = types.StringValue(peerBatch.UserId)
	data.Hostname = types.StringValue(peerBatch.Hostname)
	data.UIVersion = types.StringValue(peerBatch.UiVersion)
	data.DNSLabel = types.StringValue(peerBatch.DnsLabel)
	data.LoginExpirationEnabled = types.BoolValue(peerBatch.LoginExpirationEnabled)
	data.LoginExpired = types.BoolValue(peerBatch.LoginExpired)
	data.LastLogin = types.StringValue(peerBatch.LastLogin.String())
	data.InactivityExpirationEnabled = types.BoolValue(peerBatch.InactivityExpirationEnabled)
	data.ApprovalRequired = types.BoolValue(peerBatch.ApprovalRequired)
	data.CountryCode = types.StringValue(peerBatch.CountryCode)
	data.CityName = types.StringValue(peerBatch.CityName)
	data.SerialNumber = types.StringValue(peerBatch.SerialNumber)
	data.ExtraDNSLabels = convertStrings(peerBatch.ExtraDnsLabels) // Convert list of strings
	data.AccessiblePeersCount = types.Int64Value(int64(peerBatch.AccessiblePeersCount))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
