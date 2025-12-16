package pool

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"

	"github.com/DO-Solutions/terraform-provider-docidr/docidr/cidr"
	"github.com/DO-Solutions/terraform-provider-docidr/docidr/config"
	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ResourceDocidrPool returns the docidr_pool resource schema.
func ResourceDocidrPool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDocidrPoolCreate,
		ReadContext:   resourceDocidrPoolRead,
		DeleteContext: resourceDocidrPoolDelete,

		// No UpdateContext - all fields are ForceNew

		Schema: poolSchema(),

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			// Validate unique allocation names
			if allocations, ok := diff.GetOk("allocation"); ok {
				if err := validateUniqueAllocationNames(allocations.([]interface{})); err != nil {
					return err
				}
			}
			return nil
		},

		Description: "Allocates non-conflicting CIDR blocks for use with DigitalOcean VPCs and Kubernetes clusters.",
	}
}

// resourceDocidrPoolCreate handles the creation of a docidr_pool resource.
func resourceDocidrPoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*config.CombinedConfig).GodoClient()

	baseCIDR := d.Get("base_cidr").(string)
	allocationRequests := expandAllocations(d.Get("allocation").([]interface{}))

	// Collect user-specified exclusions
	userExclusions, err := expandExclusions(d.Get("exclude").([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	// Collect existing CIDRs from DigitalOcean account
	existingCIDRs, err := collectExistingCIDRs(ctx, client)
	if err != nil {
		return diag.Errorf("Error querying existing CIDRs from DigitalOcean: %s", err)
	}

	log.Printf("[DEBUG] Found %d existing CIDRs in DigitalOcean account", len(existingCIDRs))
	for _, cidr := range existingCIDRs {
		log.Printf("[DEBUG]   - %s", cidr.String())
	}

	// Combine exclusions
	allExclusions := append(existingCIDRs, userExclusions...)

	// Create allocator and perform allocations
	allocator, err := cidr.NewAllocator(baseCIDR)
	if err != nil {
		return diag.Errorf("Error creating CIDR allocator: %s", err)
	}

	results, err := allocator.Allocate(allocationRequests, allExclusions)
	if err != nil {
		return diag.Errorf("Error allocating CIDRs: %s", err)
	}

	log.Printf("[DEBUG] Successfully allocated CIDRs:")
	for name, cidrBlock := range results {
		log.Printf("[DEBUG]   - %s: %s", name, cidrBlock)
	}

	// Generate a stable resource ID based on inputs
	id := generateResourceID(baseCIDR, allocationRequests, d.Get("exclude").([]interface{}))
	d.SetId(id)

	// Set computed attributes
	if err := d.Set("allocations", flattenAllocations(results)); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Created docidr_pool %s", d.Id())

	return nil
}

// resourceDocidrPoolRead handles reading a docidr_pool resource.
// Since allocations are stored in state and not in any external system,
// we simply return the current state without any API calls.
func resourceDocidrPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// State is the source of truth - no API calls needed
	log.Printf("[DEBUG] Reading docidr_pool %s from state", d.Id())
	return nil
}

// resourceDocidrPoolDelete handles deletion of a docidr_pool resource.
// Since there are no external resources to delete, we just remove from state.
func resourceDocidrPoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Deleting docidr_pool %s", d.Id())
	d.SetId("")
	return nil
}

// collectExistingCIDRs queries the DigitalOcean API for all CIDRs currently in use.
func collectExistingCIDRs(ctx context.Context, client *godo.Client) ([]*net.IPNet, error) {
	var cidrs []*net.IPNet

	// Collect VPC CIDRs
	vpcCIDRs, err := collectVPCCIDRs(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("error collecting VPC CIDRs: %w", err)
	}
	cidrs = append(cidrs, vpcCIDRs...)

	// Collect Kubernetes cluster CIDRs
	k8sCIDRs, err := collectKubernetesCIDRs(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("error collecting Kubernetes CIDRs: %w", err)
	}
	cidrs = append(cidrs, k8sCIDRs...)

	return cidrs, nil
}

// collectVPCCIDRs retrieves all VPC IP ranges from the DigitalOcean account.
func collectVPCCIDRs(ctx context.Context, client *godo.Client) ([]*net.IPNet, error) {
	var cidrs []*net.IPNet

	opt := &godo.ListOptions{PerPage: 200}
	for {
		vpcs, resp, err := client.VPCs.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		for _, vpc := range vpcs {
			if vpc.IPRange != "" {
				network, err := cidr.ParseCIDR(vpc.IPRange)
				if err != nil {
					log.Printf("[WARN] Skipping invalid VPC CIDR %q from VPC %s: %v", vpc.IPRange, vpc.ID, err)
					continue
				}
				cidrs = append(cidrs, network)
				log.Printf("[DEBUG] Found VPC %s with CIDR %s", vpc.Name, vpc.IPRange)
			}
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}
		opt.Page = page + 1
	}

	return cidrs, nil
}

// collectKubernetesCIDRs retrieves all Kubernetes cluster and service subnets.
func collectKubernetesCIDRs(ctx context.Context, client *godo.Client) ([]*net.IPNet, error) {
	var cidrs []*net.IPNet

	opt := &godo.ListOptions{PerPage: 200}
	for {
		clusters, resp, err := client.Kubernetes.List(ctx, opt)
		if err != nil {
			return nil, err
		}

		for _, cluster := range clusters {
			if cluster.ClusterSubnet != "" {
				network, err := cidr.ParseCIDR(cluster.ClusterSubnet)
				if err != nil {
					log.Printf("[WARN] Skipping invalid cluster subnet %q from cluster %s: %v", cluster.ClusterSubnet, cluster.ID, err)
				} else {
					cidrs = append(cidrs, network)
					log.Printf("[DEBUG] Found Kubernetes cluster %s with cluster subnet %s", cluster.Name, cluster.ClusterSubnet)
				}
			}

			if cluster.ServiceSubnet != "" {
				network, err := cidr.ParseCIDR(cluster.ServiceSubnet)
				if err != nil {
					log.Printf("[WARN] Skipping invalid service subnet %q from cluster %s: %v", cluster.ServiceSubnet, cluster.ID, err)
				} else {
					cidrs = append(cidrs, network)
					log.Printf("[DEBUG] Found Kubernetes cluster %s with service subnet %s", cluster.Name, cluster.ServiceSubnet)
				}
			}
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}
		opt.Page = page + 1
	}

	return cidrs, nil
}

// generateResourceID creates a stable resource ID based on the configuration.
// This ensures the ID remains consistent across applies with the same inputs.
func generateResourceID(baseCIDR string, allocations []cidr.AllocationRequest, exclusions []interface{}) string {
	var parts []string

	parts = append(parts, baseCIDR)

	// Sort allocations by name for determinism
	sortedAllocs := make([]cidr.AllocationRequest, len(allocations))
	copy(sortedAllocs, allocations)
	sort.Slice(sortedAllocs, func(i, j int) bool {
		return sortedAllocs[i].Name < sortedAllocs[j].Name
	})

	for _, alloc := range sortedAllocs {
		parts = append(parts, fmt.Sprintf("%s:%d", alloc.Name, alloc.PrefixLength))
	}

	// Sort exclusions for determinism
	var exclCIDRs []string
	for _, excl := range exclusions {
		m := excl.(map[string]interface{})
		exclCIDRs = append(exclCIDRs, m["cidr"].(string))
	}
	sort.Strings(exclCIDRs)
	parts = append(parts, exclCIDRs...)

	// Create hash
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:])[:16]
}
