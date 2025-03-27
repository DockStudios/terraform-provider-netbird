resource "netbird_network" "this" {
  name = "example"
}

data "netbird_peers" "this" {
  name = "netbird-gw"
}

resource "netbird_network_router" "this" {
  network_id = netbird_network.this.id
  metric     = 500
  enabled    = true
  masquerade = true
  peer       = data.netbird_peers.this.peers[0].id
}