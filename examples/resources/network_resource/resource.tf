resource "netbird_network" "this" {
  name = "example"
}

data "netbird_peers" "this" {
  name = "netbird-gw"
}

resource "netbird_group" "this" {
  name = "Test Group"

  lifecycle {
    ignore_changes = [resources]
  }
}

resource "netbird_network_resource" "this" {
  network_id  = netbird_network.this.id
  name        = "Example"
  description = "Example Resource"
  address     = "example.com"
  peer_groups = netbird.group.this.id
  enabled     = true
}