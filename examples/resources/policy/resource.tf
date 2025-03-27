resource "netbird_group" "source" {
  name = "Source"
}

resource "netbird_group" "dest" {
  name = "Dest"
}

resource "netbird_policy" "this" {
  name    = "Example policy"
  enabled = true
  rules = [
    {
      name          = "Allow port 80 to 81"
      enabled       = true
      action        = "accept"
      bidirectional = true
      protocol      = "tcp"
      port_ranges = [
        {
          start = 80
          end   = 81
        }
      ]
      sources      = [netbird_group.source.id]
      destinations = [netbird_group.dest.id]
    }
  ]
}