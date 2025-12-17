---
page_title: "Provider: docidr"
description: |-
  The docidr provider automatically allocates non-conflicting CIDR blocks for use with DigitalOcean infrastructure.
---

# docidr Provider

The docidr provider enables automatic allocation of non-conflicting CIDR blocks for use with DigitalOcean VPCs, Kubernetes clusters, and other network-dependent resources.

When managing multiple environments or stacks via Terraform, you must typically track which CIDR ranges are in use and manually select non-conflicting values. This becomes error-prone at scale. The docidr provider solves this by querying existing network allocations within your DigitalOcean account and computing available CIDR ranges automatically.

## Example Usage

```terraform
terraform {
  required_providers {
    docidr = {
      source = "DO-Solutions/docidr"
    }
    digitalocean = {
      source = "digitalocean/digitalocean"
    }
  }
}

provider "docidr" {
  # Uses DIGITALOCEAN_TOKEN environment variable by default
}

provider "digitalocean" {
  # Uses DIGITALOCEAN_TOKEN environment variable by default
}

resource "docidr_pool" "network" {
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

resource "digitalocean_vpc" "main" {
  name     = "production-vpc"
  region   = "nyc1"
  ip_range = docidr_pool.network.allocations.main_vpc
}

resource "digitalocean_kubernetes_cluster" "app" {
  name           = "app-cluster"
  region         = "nyc1"
  version        = "1.28.2-do.0"
  vpc_uuid       = digitalocean_vpc.main.id
  cluster_subnet = docidr_pool.network.allocations.doks_cluster
  service_subnet = docidr_pool.network.allocations.doks_services

  node_pool {
    name       = "default"
    size       = "s-2vcpu-4gb"
    node_count = 3
  }
}
```

## Authentication

The docidr provider requires a DigitalOcean API token to query existing network resources. The token can be provided in the following ways:

### Environment Variable (Recommended)

Set the `DIGITALOCEAN_TOKEN` or `DIGITALOCEAN_ACCESS_TOKEN` environment variable:

```shell
export DIGITALOCEAN_TOKEN="your-api-token"
# or
export DIGITALOCEAN_ACCESS_TOKEN="your-api-token"
```

### Provider Configuration

Alternatively, configure the token directly in the provider block:

```terraform
provider "docidr" {
  token = var.do_token
}
```

~> **Warning:** Hardcoding tokens in configuration files is not recommended. Use environment variables or a secrets manager.

## Argument Reference

The following arguments are supported:

* `token` - (Optional) The DigitalOcean API token. Can also be set via the `DIGITALOCEAN_TOKEN` or `DIGITALOCEAN_ACCESS_TOKEN` environment variable.

* `api_endpoint` - (Optional) The URL for the DigitalOcean API. Defaults to `https://api.digitalocean.com`. Can also be set via the `DIGITALOCEAN_API_URL` environment variable.

* `http_retry_max` - (Optional) Maximum number of retries for failed API requests. Defaults to `4`.

* `http_retry_wait_min` - (Optional) Minimum wait time in seconds between retries. Defaults to `1.0`.

* `http_retry_wait_max` - (Optional) Maximum wait time in seconds between retries. Defaults to `30.0`.
