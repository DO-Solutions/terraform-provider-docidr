package pool

import (
	"testing"

	"github.com/DO-Solutions/terraform-provider-docidr/docidr/cidr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func TestValidateUniqueAllocationNames(t *testing.T) {
	tests := []struct {
		name        string
		allocations []interface{}
		wantErr     bool
	}{
		{
			name: "unique names",
			allocations: []interface{}{
				map[string]interface{}{"name": "vpc", "prefix_length": 16},
				map[string]interface{}{"name": "cluster", "prefix_length": 20},
				map[string]interface{}{"name": "services", "prefix_length": 20},
			},
			wantErr: false,
		},
		{
			name: "duplicate names",
			allocations: []interface{}{
				map[string]interface{}{"name": "duplicate", "prefix_length": 16},
				map[string]interface{}{"name": "duplicate", "prefix_length": 20},
			},
			wantErr: true,
		},
		{
			name:        "empty allocations",
			allocations: []interface{}{},
			wantErr:     false,
		},
		{
			name: "single allocation",
			allocations: []interface{}{
				map[string]interface{}{"name": "only_one", "prefix_length": 16},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUniqueAllocationNames(tt.allocations)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUniqueAllocationNames() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if _, ok := err.(*DuplicateNameError); !ok {
					t.Errorf("expected DuplicateNameError, got %T", err)
				}
			}
		})
	}
}

func TestPrefixLengthValidation(t *testing.T) {
	validateFunc := validation.IntBetween(16, 28)

	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"valid minimum (16)", 16, false},
		{"valid maximum (28)", 28, false},
		{"valid middle (24)", 24, false},
		{"invalid below range (8)", 8, true},
		{"invalid below range (15)", 15, true},
		{"invalid above range (29)", 29, true},
		{"invalid above range (32)", 32, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := validateFunc(tt.value, "prefix_length")
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("IntBetween(16, 28)(%d) errors = %v, wantErr %v", tt.value, errs, tt.wantErr)
			}
		})
	}
}

func TestCIDRValidation(t *testing.T) {
	validateFunc := validation.IsCIDR

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid /8 CIDR", "10.0.0.0/8", false},
		{"valid /16 CIDR", "172.16.0.0/16", false},
		{"valid /24 CIDR", "192.168.1.0/24", false},
		{"invalid - not a CIDR", "not-a-cidr", true},
		{"invalid - missing prefix", "10.0.0.0", true},
		{"invalid - bad IP", "300.0.0.0/8", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errs := validateFunc(tt.value, "cidr")
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("IsCIDR(%q) errors = %v, wantErr %v", tt.value, errs, tt.wantErr)
			}
		})
	}
}

func TestExpandAllocations(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"name": "vpc", "prefix_length": 16},
		map[string]interface{}{"name": "cluster", "prefix_length": 20},
	}

	result := expandAllocations(input)

	if len(result) != 2 {
		t.Fatalf("expected 2 allocations, got %d", len(result))
	}

	if result[0].Name != "vpc" || result[0].PrefixLength != 16 {
		t.Errorf("first allocation = %+v, want {Name: vpc, PrefixLength: 16}", result[0])
	}

	if result[1].Name != "cluster" || result[1].PrefixLength != 20 {
		t.Errorf("second allocation = %+v, want {Name: cluster, PrefixLength: 20}", result[1])
	}
}

func TestExpandAllocations_Empty(t *testing.T) {
	result := expandAllocations([]interface{}{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestExpandExclusions(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"cidr": "10.0.0.0/16", "reason": "reserved"},
		map[string]interface{}{"cidr": "172.16.0.0/12", "reason": ""},
	}

	result, err := expandExclusions(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 exclusions, got %d", len(result))
	}

	if result[0].String() != "10.0.0.0/16" {
		t.Errorf("first exclusion = %s, want 10.0.0.0/16", result[0].String())
	}

	if result[1].String() != "172.16.0.0/12" {
		t.Errorf("second exclusion = %s, want 172.16.0.0/12", result[1].String())
	}
}

func TestExpandExclusions_InvalidCIDR(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"cidr": "invalid-cidr", "reason": "test"},
	}

	_, err := expandExclusions(input)
	if err == nil {
		t.Error("expected error for invalid CIDR, got nil")
	}
}

func TestExpandExclusions_Empty(t *testing.T) {
	result, err := expandExclusions([]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestFlattenAllocations(t *testing.T) {
	input := map[string]string{
		"vpc":     "10.0.0.0/16",
		"cluster": "10.1.0.0/20",
	}

	result := flattenAllocations(input)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	if result["vpc"] != "10.0.0.0/16" {
		t.Errorf("vpc = %v, want 10.0.0.0/16", result["vpc"])
	}

	if result["cluster"] != "10.1.0.0/20" {
		t.Errorf("cluster = %v, want 10.1.0.0/20", result["cluster"])
	}
}

func TestFlattenAllocations_Empty(t *testing.T) {
	result := flattenAllocations(map[string]string{})
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d items", len(result))
	}
}

func TestPoolSchema(t *testing.T) {
	s := poolSchema()

	// Verify required fields exist
	requiredFields := []string{"allocation", "allocations"}
	for _, field := range requiredFields {
		if _, ok := s[field]; !ok {
			t.Errorf("schema missing required field: %s", field)
		}
	}

	// Verify optional fields exist
	optionalFields := []string{"base_cidr", "exclude"}
	for _, field := range optionalFields {
		if _, ok := s[field]; !ok {
			t.Errorf("schema missing optional field: %s", field)
		}
	}

	// Verify allocation is Required and ForceNew
	if !s["allocation"].Required {
		t.Error("allocation should be Required")
	}
	if !s["allocation"].ForceNew {
		t.Error("allocation should be ForceNew")
	}

	// Verify base_cidr has correct default
	if s["base_cidr"].Default != "10.0.0.0/8" {
		t.Errorf("base_cidr default = %v, want 10.0.0.0/8", s["base_cidr"].Default)
	}

	// Verify allocations is Computed
	if !s["allocations"].Computed {
		t.Error("allocations should be Computed")
	}
}

func TestDuplicateNameError(t *testing.T) {
	err := &DuplicateNameError{Name: "test_name"}
	expected := "duplicate allocation name: test_name"
	if err.Error() != expected {
		t.Errorf("DuplicateNameError.Error() = %q, want %q", err.Error(), expected)
	}
}

// Verify schema types are correct
func TestPoolSchemaTypes(t *testing.T) {
	s := poolSchema()

	typeTests := []struct {
		field    string
		expected schema.ValueType
	}{
		{"allocation", schema.TypeList},
		{"base_cidr", schema.TypeString},
		{"exclude", schema.TypeList},
		{"allocations", schema.TypeMap},
	}

	for _, tt := range typeTests {
		t.Run(tt.field, func(t *testing.T) {
			if s[tt.field].Type != tt.expected {
				t.Errorf("%s type = %v, want %v", tt.field, s[tt.field].Type, tt.expected)
			}
		})
	}
}

// Suppress unused import errors
var _ = cidr.AllocationRequest{}
