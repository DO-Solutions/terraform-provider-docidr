terraform {
  required_providers {
    docidr = {
      source = "github.com/DO-Solutions/docidr"
    }
    digitalocean = {
      source = "digitalocean/digitalocean"
    }
  }
}

# Both providers use DIGITALOCEAN_TOKEN environment variable
provider "docidr" {}

provider "digitalocean" {}

# Allocate non-conflicting CIDRs for our infrastructure
resource "docidr_pool" "network" {
  # Allocate from the default 10.0.0.0/8 range

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

# Create VPC using the allocated CIDR
resource "digitalocean_vpc" "main" {
  name     = "example-vpc"
  region   = "nyc1"
  ip_range = docidr_pool.network.allocations.main_vpc
}

# Create Kubernetes cluster using the allocated CIDRs
resource "digitalocean_kubernetes_cluster" "app" {
  name           = "example-cluster"
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

# Output the allocated CIDRs
output "vpc_cidr" {
  description = "The CIDR block allocated for the VPC"
  value       = docidr_pool.network.allocations.main_vpc
}

output "cluster_cidr" {
  description = "The CIDR block allocated for Kubernetes cluster subnet"
  value       = docidr_pool.network.allocations.doks_cluster
}

output "services_cidr" {
  description = "The CIDR block allocated for Kubernetes services subnet"
  value       = docidr_pool.network.allocations.doks_services
}

output "all_allocations" {
  description = "All allocated CIDR blocks"
  value       = docidr_pool.network.allocations
}
