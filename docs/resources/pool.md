---
page_title: "docidr_pool Resource - docidr"
subcategory: ""
description: |-
  Allocates non-conflicting CIDR blocks for use with DigitalOcean VPCs and Kubernetes clusters.
---

# docidr_pool (Resource)

Allocates non-conflicting CIDR blocks for use with DigitalOcean VPCs and Kubernetes clusters.

This resource queries existing network allocations within your DigitalOcean account (VPCs and Kubernetes cluster subnets) and computes available CIDR ranges that don't conflict with existing infrastructure.

## Example Usage

### Basic Usage

```terraform
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
```

### Custom Base CIDR

```terraform
resource "docidr_pool" "network" {
  base_cidr = "172.16.0.0/12"

  allocation {
    name          = "vpc"
    prefix_length = 16
  }
}
```

### With Exclusions

```terraform
resource "docidr_pool" "network" {
  exclude {
    cidr   = "10.255.0.0/16"
    reason = "Reserved for VPN connectivity"
  }

  exclude {
    cidr   = "10.0.0.0/16"
    reason = "Legacy network - do not use"
  }

  allocation {
    name          = "main_vpc"
    prefix_length = 16
  }
}
```

### Complete VPC and Kubernetes Setup

```terraform
resource "docidr_pool" "production" {
  allocation {
    name          = "vpc"
    prefix_length = 16
  }

  allocation {
    name          = "k8s_cluster"
    prefix_length = 20
  }

  allocation {
    name          = "k8s_services"
    prefix_length = 20
  }
}

resource "digitalocean_vpc" "production" {
  name     = "production"
  region   = "nyc1"
  ip_range = docidr_pool.production.allocations.vpc
}

resource "digitalocean_kubernetes_cluster" "production" {
  name           = "production"
  region         = "nyc1"
  version        = "1.28.2-do.0"
  vpc_uuid       = digitalocean_vpc.production.id
  cluster_subnet = docidr_pool.production.allocations.k8s_cluster
  service_subnet = docidr_pool.production.allocations.k8s_services

  node_pool {
    name       = "default"
    size       = "s-2vcpu-4gb"
    node_count = 3
  }
}
```

## Argument Reference

The following arguments are supported:

### allocation (Required, Block)

One or more `allocation` blocks defining CIDR allocation requests. Each block supports:

* `name` - (Required) Unique identifier for this allocation. Used as the key in the `allocations` output map. Must start with a letter and contain only letters, numbers, and underscores.

* `prefix_length` - (Required) The size of the CIDR block to allocate, specified as the prefix length (e.g., `24` for a /24 block). Valid range: 16-28 per DigitalOcean VPC requirements.

### base_cidr (Optional)

The parent CIDR range from which allocations are made. All allocated blocks will be subnets of this range. Defaults to `10.0.0.0/8`.

### exclude (Optional, Block)

Zero or more `exclude` blocks defining CIDR ranges to exclude from allocation. Each block supports:

* `cidr` - (Required) A CIDR range to exclude from allocation.

* `reason` - (Optional) Documentation field explaining why this range is excluded.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - A unique identifier for the resource instance.

* `allocations` - A map from allocation names to their assigned CIDR blocks. Access individual allocations using dot notation: `docidr_pool.network.allocations.main_vpc`.

## Behavior

### Allocation Algorithm

The resource allocates CIDRs sequentially from the beginning of `base_cidr`:

1. Queries all existing VPC IP ranges and Kubernetes cluster/service subnets
2. Combines these with user-specified exclusions
3. For each allocation request (in declaration order), finds the first available block that doesn't overlap with any existing or previously allocated CIDR
4. Stores all allocations in Terraform state

### State Persistence

Allocated CIDRs are stored in Terraform state and remain stable across `terraform apply` runs. The resource does not re-query the DigitalOcean API during read operations - state is the source of truth.

### ForceNew Behavior

This resource uses full replacement semantics. Any change to the following will force replacement of the entire resource:

- Adding, removing, or modifying any `allocation` block
- Changing `base_cidr`
- Adding, removing, or modifying any `exclude` block

~> **Note:** Replacing this resource will cause all dependent resources (VPCs, Kubernetes clusters) to show as requiring updates in the plan.

### Conflict Detection

The resource queries existing allocations only during creation. It does not detect conflicts that occur outside of Terraform after initial creation.

## Import

This resource does not support import, as the allocations are computed values that cannot be reconstructed from external state.
