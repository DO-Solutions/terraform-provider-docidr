package pool

import (
	"net"
	"regexp"

	"github.com/DO-Solutions/terraform-provider-docidr/docidr/cidr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// poolSchema returns the schema for the docidr_pool resource.
func poolSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"allocation": {
			Type:        schema.TypeList,
			Required:    true,
			ForceNew:    true,
			MinItems:    1,
			Description: "List of CIDR allocation requests. Each allocation specifies a name and prefix length.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:        schema.TypeString,
						Required:    true,
						ForceNew:    true,
						Description: "Unique identifier for this allocation. Used as the key in the allocations output map.",
						ValidateFunc: validation.All(
							validation.StringLenBetween(1, 64),
							validation.StringMatch(
								regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`),
								"must start with a letter and contain only letters, numbers, and underscores",
							),
						),
					},
					"prefix_length": {
						Type:         schema.TypeInt,
						Required:     true,
						ForceNew:     true,
						Description:  "The prefix length for the CIDR block (e.g., 24 for /24). Valid range: 16-28.",
						ValidateFunc: validation.IntBetween(16, 28),
					},
				},
			},
		},
		"base_cidr": {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "10.0.0.0/8",
			ForceNew:     true,
			Description:  "The parent CIDR range from which allocations are made. All allocated blocks will be subnets of this range.",
			ValidateFunc: validation.IsCIDR,
		},
		"exclude": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    true,
			Description: "List of CIDR ranges to exclude from allocation.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cidr": {
						Type:         schema.TypeString,
						Required:     true,
						ForceNew:     true,
						Description:  "A CIDR range to exclude from allocation.",
						ValidateFunc: validation.IsCIDR,
					},
					"reason": {
						Type:        schema.TypeString,
						Optional:    true,
						ForceNew:    true,
						Description: "Optional documentation explaining why this range is excluded.",
					},
				},
			},
		},
		"allocations": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "Map of allocation names to their assigned CIDR blocks.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}
}

// AllocationConfig represents an allocation request parsed from the schema.
type AllocationConfig struct {
	Name         string
	PrefixLength int
}

// ExclusionConfig represents an exclusion parsed from the schema.
type ExclusionConfig struct {
	CIDR   string
	Reason string
}

// expandAllocations converts the allocation list from the schema to AllocationConfig slice.
func expandAllocations(allocations []interface{}) []cidr.AllocationRequest {
	result := make([]cidr.AllocationRequest, 0, len(allocations))
	for _, alloc := range allocations {
		m := alloc.(map[string]interface{})
		result = append(result, cidr.AllocationRequest{
			Name:         m["name"].(string),
			PrefixLength: m["prefix_length"].(int),
		})
	}
	return result
}

// expandExclusions converts the exclude list from the schema to a slice of net.IPNet.
func expandExclusions(exclusions []interface{}) ([]*net.IPNet, error) {
	result := make([]*net.IPNet, 0, len(exclusions))
	for _, excl := range exclusions {
		m := excl.(map[string]interface{})
		cidrStr := m["cidr"].(string)
		network, err := cidr.ParseCIDR(cidrStr)
		if err != nil {
			return nil, err
		}
		result = append(result, network)
	}
	return result, nil
}

// flattenAllocations converts the allocation results map to a schema-compatible format.
func flattenAllocations(allocations map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for name, cidrBlock := range allocations {
		result[name] = cidrBlock
	}
	return result
}

// validateUniqueAllocationNames checks that all allocation names are unique.
func validateUniqueAllocationNames(allocations []interface{}) error {
	seen := make(map[string]bool)
	for _, alloc := range allocations {
		m := alloc.(map[string]interface{})
		name := m["name"].(string)
		if seen[name] {
			return &DuplicateNameError{Name: name}
		}
		seen[name] = true
	}
	return nil
}

// DuplicateNameError is returned when duplicate allocation names are found.
type DuplicateNameError struct {
	Name string
}

func (e *DuplicateNameError) Error() string {
	return "duplicate allocation name: " + e.Name
}
