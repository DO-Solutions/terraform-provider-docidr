package cidr

import (
	"encoding/binary"
	"fmt"
	"net"
)

// AllocationRequest represents a request to allocate a CIDR block.
type AllocationRequest struct {
	Name         string
	PrefixLength int
}

// Allocator handles CIDR block allocation within a base range.
type Allocator struct {
	baseCIDR *net.IPNet
}

// NewAllocator creates a new CIDR allocator for the given base CIDR.
func NewAllocator(baseCIDR string) (*Allocator, error) {
	_, network, err := net.ParseCIDR(baseCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid base CIDR %q: %w", baseCIDR, err)
	}

	return &Allocator{
		baseCIDR: network,
	}, nil
}

// Allocate finds available CIDR blocks for each request, avoiding the given exclusions.
// Allocations are made sequentially, with each new allocation added to the exclusion
// list before processing the next request.
func (a *Allocator) Allocate(requests []AllocationRequest, exclusions []*net.IPNet) (map[string]string, error) {
	results := make(map[string]string)

	// Copy exclusions to avoid modifying the original slice
	usedBlocks := make([]*net.IPNet, len(exclusions))
	copy(usedBlocks, exclusions)

	for _, req := range requests {
		// Validate prefix length is within base CIDR
		basePrefixLen, _ := a.baseCIDR.Mask.Size()
		if req.PrefixLength < basePrefixLen {
			return nil, fmt.Errorf("requested prefix length /%d for %q is smaller than base CIDR prefix /%d",
				req.PrefixLength, req.Name, basePrefixLen)
		}

		allocated, err := a.findAvailableBlock(req.PrefixLength, usedBlocks)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate CIDR for %q (/%d): %w", req.Name, req.PrefixLength, err)
		}

		results[req.Name] = allocated.String()
		usedBlocks = append(usedBlocks, allocated)
	}

	return results, nil
}

// findAvailableBlock finds the first available CIDR block of the given prefix length
// that doesn't overlap with any of the exclusions.
func (a *Allocator) findAvailableBlock(prefixLen int, exclusions []*net.IPNet) (*net.IPNet, error) {
	// Create mask for the requested prefix length
	mask := net.CIDRMask(prefixLen, 32)

	// Start from the beginning of the base CIDR
	currentIP := a.baseCIDR.IP.Mask(a.baseCIDR.Mask)

	// Calculate the block size for the requested prefix
	blockSize := uint32(1) << (32 - prefixLen)

	// Convert base CIDR boundaries to uint32 for easier math
	baseStart := ipToUint32(a.baseCIDR.IP.Mask(a.baseCIDR.Mask))
	basePrefixLen, _ := a.baseCIDR.Mask.Size()
	baseEnd := baseStart + (uint32(1) << (32 - basePrefixLen))

	// Start scanning from the beginning
	candidateStart := baseStart

	// Align to block boundary
	if candidateStart%blockSize != 0 {
		candidateStart = ((candidateStart / blockSize) + 1) * blockSize
	}

	for candidateStart+blockSize <= baseEnd {
		candidate := &net.IPNet{
			IP:   uint32ToIP(candidateStart),
			Mask: mask,
		}

		// Check if candidate overlaps with any exclusion
		overlaps := false
		for _, exclusion := range exclusions {
			if networksOverlap(candidate, exclusion) {
				overlaps = true
				// Skip past the overlapping exclusion
				exclStart := ipToUint32(exclusion.IP.Mask(exclusion.Mask))
				exclPrefixLen, _ := exclusion.Mask.Size()
				exclEnd := exclStart + (uint32(1) << (32 - exclPrefixLen))

				// Move candidate past the exclusion, aligned to block boundary
				candidateStart = exclEnd
				if candidateStart%blockSize != 0 {
					candidateStart = ((candidateStart / blockSize) + 1) * blockSize
				}
				break
			}
		}

		if !overlaps {
			return candidate, nil
		}
	}

	return nil, fmt.Errorf("no available space for /%d block in %s (tried from %s)",
		prefixLen, a.baseCIDR.String(), currentIP.String())
}

// networksOverlap returns true if two CIDR blocks overlap.
func networksOverlap(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
}

// ipToUint32 converts an IPv4 address to a uint32.
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}

// uint32ToIP converts a uint32 to an IPv4 address.
func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

// ParseCIDR parses a CIDR string and returns the network.
func ParseCIDR(cidr string) (*net.IPNet, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	return network, nil
}

// ParseCIDRs parses multiple CIDR strings and returns the networks.
func ParseCIDRs(cidrs []string) ([]*net.IPNet, error) {
	networks := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		network, err := ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}
	return networks, nil
}
