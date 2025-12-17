#!/usr/bin/env bash
#
# Install script for terraform-provider-docidr
# Downloads and installs the latest release from GitHub
#

set -euo pipefail

REPO="DO-Solutions/terraform-provider-docidr"
PROVIDER_NAME="docidr"
PROVIDER_NAMESPACE="github.com/DO-Solutions"

# Detect OS
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        darwin) echo "darwin" ;;
        linux) echo "linux" ;;
        mingw*|msys*|cygwin*) echo "windows" ;;
        freebsd) echo "freebsd" ;;
        *)
            echo "Unsupported OS: $os" >&2
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        i386|i686) echo "386" ;;
        arm64|aarch64) echo "arm64" ;;
        armv*) echo "arm" ;;
        *)
            echo "Unsupported architecture: $arch" >&2
            exit 1
            ;;
    esac
}

# Get latest release version from GitHub API
get_latest_version() {
    local version
    version=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$version" ]]; then
        echo "Failed to determine latest version" >&2
        exit 1
    fi
    # Remove 'v' prefix if present
    echo "${version#v}"
}

main() {
    local os arch version install_dir download_url zip_file binary_name

    echo "Installing terraform-provider-docidr..."

    os=$(detect_os)
    arch=$(detect_arch)
    version=$(get_latest_version)

    echo "  OS: $os"
    echo "  Arch: $arch"
    echo "  Version: $version"

    install_dir="${HOME}/.terraform.d/plugins/${PROVIDER_NAMESPACE}/${PROVIDER_NAME}/${version}/${os}_${arch}"
    download_url="https://github.com/${REPO}/releases/download/v${version}/terraform-provider-${PROVIDER_NAME}_${version}_${os}_${arch}.zip"
    zip_file=$(mktemp)
    binary_name="terraform-provider-${PROVIDER_NAME}_v${version}"

    echo "  Downloading from: $download_url"

    # Download
    if ! curl -sSL -o "$zip_file" "$download_url"; then
        echo "Failed to download release" >&2
        rm -f "$zip_file"
        exit 1
    fi

    # Create install directory
    mkdir -p "$install_dir"

    # Extract
    echo "  Installing to: $install_dir"
    if command -v unzip &> /dev/null; then
        unzip -o -q "$zip_file" -d "$install_dir"
    else
        echo "unzip command not found. Please install unzip." >&2
        rm -f "$zip_file"
        exit 1
    fi

    # Cleanup
    rm -f "$zip_file"

    # Make executable (not needed on Windows)
    if [[ "$os" != "windows" ]]; then
        chmod +x "${install_dir}/${binary_name}"
    fi

    echo ""
    echo "Successfully installed terraform-provider-docidr v${version}"
    echo ""
    echo "Add the following to your Terraform configuration:"
    echo ""
    echo '  terraform {'
    echo '    required_providers {'
    echo '      docidr = {'
    echo "        source  = \"${PROVIDER_NAMESPACE}/${PROVIDER_NAME}\""
    echo "        version = \"${version}\""
    echo '      }'
    echo '    }'
    echo '  }'
    echo ""
}

main "$@"
