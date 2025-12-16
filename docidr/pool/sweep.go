package pool

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func init() {
	resource.AddTestSweepers("docidr_pool", &resource.Sweeper{
		Name: "docidr_pool",
		F:    sweepPool,
	})
}

// sweepPool cleans up test resources.
// Since docidr_pool only exists in Terraform state and has no API-side resources,
// there's nothing to clean up. This sweeper is included for consistency with
// other Terraform providers and future expansion.
func sweepPool(region string) error {
	log.Println("[DEBUG] docidr_pool sweep: No resources to clean up (state-only resource)")
	return nil
}
