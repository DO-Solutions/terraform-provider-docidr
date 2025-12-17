package config

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"golang.org/x/oauth2"
)

// Config holds the provider configuration.
type Config struct {
	Token            string
	APIEndpoint      string
	TerraformVersion string
	HTTPRetryMax     int
	HTTPRetryWaitMax float64
	HTTPRetryWaitMin float64
}

// CombinedConfig wraps the godo client for use by resources.
type CombinedConfig struct {
	client *godo.Client
}

// GodoClient returns the underlying godo client.
func (c *CombinedConfig) GodoClient() *godo.Client {
	return c.client
}

// Client creates a new godo client from the configuration.
func (c *Config) Client() (*CombinedConfig, error) {
	tokenSrc := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: c.Token,
	})

	userAgent := fmt.Sprintf("Terraform/%s", c.TerraformVersion)
	var godoOpts []godo.ClientOpt

	client := oauth2.NewClient(context.Background(), tokenSrc)

	if c.HTTPRetryMax > 0 {
		retryConfig := godo.RetryConfig{
			RetryMax:     c.HTTPRetryMax,
			RetryWaitMin: godo.PtrTo(c.HTTPRetryWaitMin),
			RetryWaitMax: godo.PtrTo(c.HTTPRetryWaitMax),
			Logger:       log.Default(),
		}

		godoOpts = []godo.ClientOpt{godo.WithRetryAndBackoffs(retryConfig)}
	}

	godoOpts = append(godoOpts, godo.SetUserAgent(userAgent))

	godoClient, err := godo.New(client, godoOpts...)
	if err != nil {
		return nil, err
	}

	// Add logging transport for debugging
	// TODO: logging.NewTransport is deprecated and should be replaced with
	// logging.NewTransportWithRequestLogging.
	//
	//nolint:staticcheck
	clientTransport := logging.NewTransport("DigitalOcean", godoClient.HTTPClient.Transport)
	godoClient.HTTPClient.Transport = clientTransport

	if c.APIEndpoint != "" {
		apiURL, err := url.Parse(c.APIEndpoint)
		if err != nil {
			return nil, err
		}
		godoClient.BaseURL = apiURL
	}

	log.Printf("[INFO] DigitalOcean Client configured for URL: %s", godoClient.BaseURL.String())

	return &CombinedConfig{
		client: godoClient,
	}, nil
}

// DefaultHTTPClient returns a basic HTTP client for simple API calls.
func DefaultHTTPClient(token string) *http.Client {
	tokenSrc := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	return oauth2.NewClient(context.Background(), tokenSrc)
}
