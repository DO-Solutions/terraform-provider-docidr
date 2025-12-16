package cidr

import (
	"net"
	"testing"
)

func TestNewAllocator(t *testing.T) {
	tests := []struct {
		name     string
		baseCIDR string
		wantErr  bool
	}{
		{
			name:     "valid /8 CIDR",
			baseCIDR: "10.0.0.0/8",
			wantErr:  false,
		},
		{
			name:     "valid /16 CIDR",
			baseCIDR: "172.16.0.0/16",
			wantErr:  false,
		},
		{
			name:     "invalid CIDR",
			baseCIDR: "not-a-cidr",
			wantErr:  true,
		},
		{
			name:     "missing prefix",
			baseCIDR: "10.0.0.0",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAllocator(tt.baseCIDR)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAllocator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAllocator_Allocate_Basic(t *testing.T) {
	allocator, err := NewAllocator("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	requests := []AllocationRequest{
		{Name: "vpc", PrefixLength: 16},
		{Name: "cluster", PrefixLength: 20},
		{Name: "services", PrefixLength: 20},
	}

	results, err := allocator.Allocate(requests, nil)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	// Verify expected allocations
	expected := map[string]string{
		"vpc":      "10.0.0.0/16",
		"cluster":  "10.1.0.0/20",
		"services": "10.1.16.0/20",
	}

	for name, expectedCIDR := range expected {
		if results[name] != expectedCIDR {
			t.Errorf("Allocation %q = %v, want %v", name, results[name], expectedCIDR)
		}
	}
}

func TestAllocator_Allocate_WithExclusions(t *testing.T) {
	allocator, err := NewAllocator("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	// Exclude the first /16 block
	exclusions := []*net.IPNet{
		mustParseCIDR("10.0.0.0/16"),
	}

	requests := []AllocationRequest{
		{Name: "vpc", PrefixLength: 16},
	}

	results, err := allocator.Allocate(requests, exclusions)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	// Should skip 10.0.0.0/16 and allocate 10.1.0.0/16
	if results["vpc"] != "10.1.0.0/16" {
		t.Errorf("vpc = %v, want 10.1.0.0/16", results["vpc"])
	}
}

func TestAllocator_Allocate_MultipleExclusions(t *testing.T) {
	allocator, err := NewAllocator("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	// Exclude first three /16 blocks
	exclusions := []*net.IPNet{
		mustParseCIDR("10.0.0.0/16"),
		mustParseCIDR("10.1.0.0/16"),
		mustParseCIDR("10.2.0.0/16"),
	}

	requests := []AllocationRequest{
		{Name: "vpc", PrefixLength: 16},
	}

	results, err := allocator.Allocate(requests, exclusions)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	// Should allocate 10.3.0.0/16
	if results["vpc"] != "10.3.0.0/16" {
		t.Errorf("vpc = %v, want 10.3.0.0/16", results["vpc"])
	}
}

func TestAllocator_Allocate_MixedPrefixLengths(t *testing.T) {
	allocator, err := NewAllocator("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	requests := []AllocationRequest{
		{Name: "large", PrefixLength: 16},
		{Name: "medium", PrefixLength: 20},
		{Name: "small", PrefixLength: 24},
	}

	results, err := allocator.Allocate(requests, nil)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	// Verify allocations don't overlap
	cidrs := make([]*net.IPNet, 0, len(results))
	for _, cidr := range results {
		cidrs = append(cidrs, mustParseCIDR(cidr))
	}

	for i := 0; i < len(cidrs); i++ {
		for j := i + 1; j < len(cidrs); j++ {
			if networksOverlap(cidrs[i], cidrs[j]) {
				t.Errorf("Allocations overlap: %s and %s", cidrs[i], cidrs[j])
			}
		}
	}
}

func TestAllocator_Allocate_ExhaustedSpace(t *testing.T) {
	// Use a small base CIDR
	allocator, err := NewAllocator("10.0.0.0/24")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	// Try to allocate more than available
	requests := []AllocationRequest{
		{Name: "first", PrefixLength: 25},
		{Name: "second", PrefixLength: 25},
		{Name: "third", PrefixLength: 25}, // No space left
	}

	_, err = allocator.Allocate(requests, nil)
	if err == nil {
		t.Error("Allocate() should have returned an error for exhausted space")
	}
}

func TestAllocator_Allocate_PrefixTooSmall(t *testing.T) {
	allocator, err := NewAllocator("10.0.0.0/16")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	// Request a /8 from a /16 base - should fail
	requests := []AllocationRequest{
		{Name: "too_big", PrefixLength: 8},
	}

	_, err = allocator.Allocate(requests, nil)
	if err == nil {
		t.Error("Allocate() should have returned an error for prefix smaller than base")
	}
}

func TestAllocator_Allocate_AdjacentBlocks(t *testing.T) {
	allocator, err := NewAllocator("10.0.0.0/16")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	requests := []AllocationRequest{
		{Name: "first", PrefixLength: 24},
		{Name: "second", PrefixLength: 24},
		{Name: "third", PrefixLength: 24},
	}

	results, err := allocator.Allocate(requests, nil)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	expected := map[string]string{
		"first":  "10.0.0.0/24",
		"second": "10.0.1.0/24",
		"third":  "10.0.2.0/24",
	}

	for name, expectedCIDR := range expected {
		if results[name] != expectedCIDR {
			t.Errorf("Allocation %q = %v, want %v", name, results[name], expectedCIDR)
		}
	}
}

func TestAllocator_Allocate_SkipPartialOverlap(t *testing.T) {
	allocator, err := NewAllocator("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	// Exclude a smaller block within the first /16
	exclusions := []*net.IPNet{
		mustParseCIDR("10.0.0.0/24"),
	}

	requests := []AllocationRequest{
		{Name: "vpc", PrefixLength: 16},
	}

	results, err := allocator.Allocate(requests, exclusions)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	// Should skip 10.0.0.0/16 (overlaps with exclusion) and allocate 10.1.0.0/16
	if results["vpc"] != "10.1.0.0/16" {
		t.Errorf("vpc = %v, want 10.1.0.0/16", results["vpc"])
	}
}

func TestAllocator_Allocate_EmptyRequests(t *testing.T) {
	allocator, err := NewAllocator("10.0.0.0/8")
	if err != nil {
		t.Fatalf("NewAllocator() error = %v", err)
	}

	results, err := allocator.Allocate(nil, nil)
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected empty results, got %v", results)
	}
}

func TestNetworksOverlap(t *testing.T) {
	tests := []struct {
		name    string
		a       string
		b       string
		overlap bool
	}{
		{
			name:    "identical networks",
			a:       "10.0.0.0/24",
			b:       "10.0.0.0/24",
			overlap: true,
		},
		{
			name:    "a contains b",
			a:       "10.0.0.0/16",
			b:       "10.0.1.0/24",
			overlap: true,
		},
		{
			name:    "b contains a",
			a:       "10.0.1.0/24",
			b:       "10.0.0.0/16",
			overlap: true,
		},
		{
			name:    "adjacent networks",
			a:       "10.0.0.0/24",
			b:       "10.0.1.0/24",
			overlap: false,
		},
		{
			name:    "completely separate",
			a:       "10.0.0.0/16",
			b:       "10.1.0.0/16",
			overlap: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			netA := mustParseCIDR(tt.a)
			netB := mustParseCIDR(tt.b)

			if got := networksOverlap(netA, netB); got != tt.overlap {
				t.Errorf("networksOverlap(%s, %s) = %v, want %v", tt.a, tt.b, got, tt.overlap)
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{
			name:    "valid CIDR",
			cidr:    "10.0.0.0/16",
			wantErr: false,
		},
		{
			name:    "invalid CIDR",
			cidr:    "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCIDR(tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCIDR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseCIDRs(t *testing.T) {
	tests := []struct {
		name    string
		cidrs   []string
		wantLen int
		wantErr bool
	}{
		{
			name:    "valid CIDRs",
			cidrs:   []string{"10.0.0.0/16", "172.16.0.0/16"},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "empty list",
			cidrs:   []string{},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "one invalid",
			cidrs:   []string{"10.0.0.0/16", "invalid"},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCIDRs(tt.cidrs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCIDRs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("ParseCIDRs() returned %d networks, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// mustParseCIDR parses a CIDR string or panics.
func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return network
}
