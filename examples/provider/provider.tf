terraform {
  required_providers {
    netbird = {
      source = "dockstudios/netbird"
    }
  }
}

provider "netbird" {
  # Leave this empty to default to api.netbird.io
  endpoint = "https://netbird.myorg.com"

  # Personal access token
  access_token = "nbp_abcdef1234556"

  # or Oauth2 bearer token
  # bearer_token = "nbp_abcdef"
}
