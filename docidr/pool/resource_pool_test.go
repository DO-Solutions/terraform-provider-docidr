package pool_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/DO-Solutions/terraform-provider-docidr/docidr/acceptance"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDocidrPool_Basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acceptance.TestAccPreCheck(t) },
		ProviderFactories: acceptance.TestAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDocidrPoolConfig_Basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docidr_pool.test", "base_cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "id"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.main_vpc"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.doks_cluster"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.doks_services"),
				),
			},
		},
	})
}

func TestAccDocidrPool_CustomBaseCIDR(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acceptance.TestAccPreCheck(t) },
		ProviderFactories: acceptance.TestAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDocidrPoolConfig_CustomBaseCIDR(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docidr_pool.test", "base_cidr", "172.16.0.0/12"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.vpc"),
					resource.TestMatchResourceAttr("docidr_pool.test", "allocations.vpc", regexp.MustCompile(`^172\.\d+\.\d+\.\d+/16$`)),
				),
			},
		},
	})
}

func TestAccDocidrPool_WithExclusions(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acceptance.TestAccPreCheck(t) },
		ProviderFactories: acceptance.TestAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDocidrPoolConfig_WithExclusions(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docidr_pool.test", "base_cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.vpc"),
					// The allocated CIDR should be in 10.x.x.x/16 format
					resource.TestMatchResourceAttr("docidr_pool.test", "allocations.vpc", regexp.MustCompile(`^10\.\d+\.\d+\.\d+/16$`)),
					// Verify it's not the excluded range by checking it's set (exclusion is validated in unit tests)
					testAccCheckAllocationNotEqual("docidr_pool.test", "allocations.vpc", "10.0.0.0/16"),
				),
			},
		},
	})
}

func TestAccDocidrPool_SingleAllocation(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acceptance.TestAccPreCheck(t) },
		ProviderFactories: acceptance.TestAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDocidrPoolConfig_SingleAllocation(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docidr_pool.test", "base_cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.only_vpc"),
				),
			},
		},
	})
}

func TestAccDocidrPool_ForceNew(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { acceptance.TestAccPreCheck(t) },
		ProviderFactories: acceptance.TestAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDocidrPoolConfig_ForceNew_Initial(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docidr_pool.test", "base_cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.vpc"),
				),
			},
			{
				Config: testAccDocidrPoolConfig_ForceNew_Updated(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("docidr_pool.test", "base_cidr", "10.0.0.0/8"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.vpc"),
					resource.TestCheckResourceAttrSet("docidr_pool.test", "allocations.extra"),
				),
			},
		},
	})
}

func testAccDocidrPoolConfig_Basic() string {
	return `
resource "docidr_pool" "test" {
  allocation {
    name          = "main_vpc"
    prefix_length = 16
  }

  allocation {
    name          = "doks_cluster"
    prefix_length = 20
  }

  allocation {
    name          = "doks_services"
    prefix_length = 20
  }
}
`
}

func testAccDocidrPoolConfig_CustomBaseCIDR() string {
	return `
resource "docidr_pool" "test" {
  base_cidr = "172.16.0.0/12"

  allocation {
    name          = "vpc"
    prefix_length = 16
  }
}
`
}

func testAccDocidrPoolConfig_WithExclusions() string {
	return `
resource "docidr_pool" "test" {
  exclude {
    cidr   = "10.0.0.0/16"
    reason = "Reserved for testing"
  }

  allocation {
    name          = "vpc"
    prefix_length = 16
  }
}
`
}

func testAccDocidrPoolConfig_SingleAllocation() string {
	return `
resource "docidr_pool" "test" {
  allocation {
    name          = "only_vpc"
    prefix_length = 16
  }
}
`
}

func testAccDocidrPoolConfig_ForceNew_Initial() string {
	return `
resource "docidr_pool" "test" {
  allocation {
    name          = "vpc"
    prefix_length = 16
  }
}
`
}

func testAccDocidrPoolConfig_ForceNew_Updated() string {
	return `
resource "docidr_pool" "test" {
  allocation {
    name          = "vpc"
    prefix_length = 16
  }

  allocation {
    name          = "extra"
    prefix_length = 20
  }
}
`
}

// testAccCheckAllocationNotEqual verifies that an allocation attribute is not equal to a specific value.
func testAccCheckAllocationNotEqual(resourceName, attrName, notExpected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		actual := rs.Primary.Attributes[attrName]
		if actual == notExpected {
			return fmt.Errorf("Attribute %s should not equal %s, but it does", attrName, notExpected)
		}

		return nil
	}
}

// Acceptance tests helper to suppress unused import error
var _ = fmt.Sprintf
