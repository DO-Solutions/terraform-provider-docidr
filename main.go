package main

import (
	"github.com/DO-Solutions/terraform-provider-docidr/docidr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: docidr.Provider,
	})
}
