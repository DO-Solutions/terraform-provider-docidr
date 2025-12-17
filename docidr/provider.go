package docidr

import (
	"context"

	"github.com/DO-Solutions/terraform-provider-docidr/docidr/config"
	"github.com/DO-Solutions/terraform-provider-docidr/docidr/pool"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns the docidr Terraform provider.
func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"DIGITALOCEAN_TOKEN",
					"DIGITALOCEAN_ACCESS_TOKEN",
				}, nil),
				Description: "The token key for API operations.",
			},
			"api_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("DIGITALOCEAN_API_URL", "https://api.digitalocean.com"),
				Description: "The URL to use for the DigitalOcean API.",
			},
			"http_retry_max": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     4,
				Description: "The maximum number of retries on a failed API request.",
			},
			"http_retry_wait_min": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Default:     1.0,
				Description: "The minimum wait time (in seconds) between failed API requests.",
			},
			"http_retry_wait_max": {
				Type:        schema.TypeFloat,
				Optional:    true,
				Default:     30.0,
				Description: "The maximum wait time (in seconds) between failed API requests.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"docidr_pool": pool.ResourceDocidrPool(),
		},

		DataSourcesMap: map[string]*schema.Resource{},
	}

	p.ConfigureContextFunc = providerConfigure(p)

	return p
}

func providerConfigure(p *schema.Provider) schema.ConfigureContextFunc {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		config := &config.Config{
			Token:            d.Get("token").(string),
			APIEndpoint:      d.Get("api_endpoint").(string),
			HTTPRetryMax:     d.Get("http_retry_max").(int),
			HTTPRetryWaitMin: d.Get("http_retry_wait_min").(float64),
			HTTPRetryWaitMax: d.Get("http_retry_wait_max").(float64),
			TerraformVersion: p.TerraformVersion,
		}

		if config.Token == "" {
			return nil, diag.Errorf("DigitalOcean token must be configured. Set the token in the provider configuration or use the DIGITALOCEAN_TOKEN environment variable.")
		}

		client, err := config.Client()
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return client, nil
	}
}
