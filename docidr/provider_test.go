package docidr

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProvider(t *testing.T) {
	p := Provider()
	if p == nil {
		t.Fatal("Provider() returned nil")
	}

	if err := p.InternalValidate(); err != nil {
		t.Fatalf("Provider internal validation failed: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func TestProvider_HasRequiredResources(t *testing.T) {
	p := Provider()

	expectedResources := []string{
		"docidr_pool",
	}

	for _, name := range expectedResources {
		if _, ok := p.ResourcesMap[name]; !ok {
			t.Errorf("Provider missing expected resource: %s", name)
		}
	}
}

func TestProvider_Schema(t *testing.T) {
	p := Provider()

	expectedSchemaKeys := []string{
		"token",
		"api_endpoint",
		"http_retry_max",
		"http_retry_wait_min",
		"http_retry_wait_max",
	}

	for _, key := range expectedSchemaKeys {
		if _, ok := p.Schema[key]; !ok {
			t.Errorf("Provider schema missing expected key: %s", key)
		}
	}
}
