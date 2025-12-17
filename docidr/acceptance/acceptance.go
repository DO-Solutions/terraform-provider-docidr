package acceptance

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/DO-Solutions/terraform-provider-docidr/docidr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestNamePrefix is the prefix for all test resource names.
const TestNamePrefix = "tf-acc-test-"

var (
	// TestAccProvider is the docidr provider instance for testing.
	TestAccProvider *schema.Provider

	// TestAccProviders is a map of provider instances for testing.
	TestAccProviders map[string]*schema.Provider

	// TestAccProviderFactories is a map of provider factory functions for testing.
	TestAccProviderFactories map[string]func() (*schema.Provider, error)
)

func init() {
	TestAccProvider = docidr.Provider()
	TestAccProviders = map[string]*schema.Provider{
		"docidr": TestAccProvider,
	}
	TestAccProviderFactories = map[string]func() (*schema.Provider, error){
		"docidr": func() (*schema.Provider, error) {
			return TestAccProvider, nil
		},
	}
}

// TestAccPreCheck validates the necessary test API keys exist in the environment.
func TestAccPreCheck(t *testing.T) {
	if os.Getenv("DIGITALOCEAN_TOKEN") == "" && os.Getenv("DIGITALOCEAN_ACCESS_TOKEN") == "" {
		t.Fatal("DIGITALOCEAN_TOKEN or DIGITALOCEAN_ACCESS_TOKEN must be set for acceptance tests")
	}

	err := TestAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
	if err != nil {
		t.Fatal(err)
	}
}

// RandomTestName generates a random name with the test prefix.
func RandomTestName(additionalNames ...string) string {
	prefix := TestNamePrefix
	for _, n := range additionalNames {
		prefix += "-" + strings.Replace(n, " ", "_", -1)
	}
	return randomName(prefix, 10)
}

func randomName(prefix string, length int) string {
	return fmt.Sprintf("%s%s", prefix, acctest.RandString(length))
}
