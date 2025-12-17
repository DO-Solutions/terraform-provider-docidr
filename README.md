# Terraform Provider for DigitalOcean CIDR Allocation

This Terraform provider automatically allocates non-conflicting CIDR blocks for use with DigitalOcean VPCs, Kubernetes clusters, and other network-dependent resources.

## Problem Statement

All CIDRs within a DigitalOcean team must be unique across VPCs, DOKS cluster subnets, and DOKS service subnets. When managing multiple environments or stacks via Terraform, users must manually track which CIDR ranges are in use and select non-conflicting values. This becomes error-prone at scale and creates friction when provisioning new infrastructure.

## Solution

The `docidr` provider queries existing network allocations within your DigitalOcean account and computes available CIDR ranges automatically, ensuring no conflicts with existing infrastructure.

## Installation

### Option 1: Install Script (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/DO-Solutions/terraform-provider-docidr/main/scripts/install.sh | bash
```

The script will:
1. Download the appropriate binary for your platform
2. Install it to the correct Terraform plugin directory
3. Configure `~/.terraformrc` with a filesystem mirror

### Option 2: Manual Installation

1. Download the appropriate release from [GitHub Releases](https://github.com/DO-Solutions/terraform-provider-docidr/releases)
2. Extract and install to the plugin directory:

```bash
# Example for Linux amd64
VERSION="0.0.1"
OS="linux"
ARCH="amd64"

unzip terraform-provider-docidr_${VERSION}_${OS}_${ARCH}.zip
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/DO-Solutions/docidr/${VERSION}/${OS}_${ARCH}
mv terraform-provider-docidr_v${VERSION} ~/.terraform.d/plugins/registry.terraform.io/DO-Solutions/docidr/${VERSION}/${OS}_${ARCH}/
```

3. Add the following to `~/.terraformrc`:

```hcl
provider_installation {
  filesystem_mirror {
    path    = "~/.terraform.d/plugins"
    include = ["DO-Solutions/docidr"]
  }
  direct {
    exclude = ["DO-Solutions/docidr"]
  }
}
```

4. Add the provider to your Terraform configuration:

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
```

## Quick Start

```terraform
provider "docidr" {
  # Uses DIGITALOCEAN_TOKEN environment variable
}

provider "digitalocean" {
  # Uses DIGITALOCEAN_TOKEN environment variable
}

# Allocate non-conflicting CIDRs
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

# Use the allocated CIDRs
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

## Features

- **Automatic allocation**: Determines available CIDRs without manual lookup
- **State persistence**: Allocated CIDRs remain stable across Terraform applies
- **Intra-resource coordination**: Multiple allocations within a single resource do not conflict with each other
- **Account-wide awareness**: Queries existing DigitalOcean VPCs and Kubernetes clusters to avoid conflicts
- **Exclusion support**: Manually exclude specific CIDR ranges (e.g., for VPN connectivity)
- **Custom base CIDR**: Allocate from any private IP range, not just 10.0.0.0/8

## Documentation

- [Provider Documentation](docs/index.md)
- [docidr_pool Resource](docs/resources/pool.md)

## Development

### Requirements

- [Go](https://golang.org/doc/install) 1.23+
- [Terraform](https://www.terraform.io/downloads.html) 1.0+
- A DigitalOcean API token

### Building

```shell
make build
```

### Testing

Run unit tests:
```shell
make test
```

Run acceptance tests (requires `DIGITALOCEAN_TOKEN`):
```shell
make testacc
```

### Linting

```shell
make lint
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## License

This `docidr` Terraform Provider is offered without support from DigitalOcean.
