package sweep

import (
	"testing"

	// Import pool package to register its sweeper
	_ "github.com/DO-Solutions/terraform-provider-docidr/docidr/pool"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}
