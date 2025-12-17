#!/usr/bin/env bash
#
# Install script for terraform-provider-docidr
# Downloads and installs the latest release from GitHub
#

set -euo pipefail

REPO="DO-Solutions/terraform-provider-docidr"
PROVIDER_NAME="docidr"
PROVIDER_SOURCE="DO-Solutions/docidr"

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

# Configure terraformrc with dev_overrides
configure_terraformrc() {
    local install_dir="$1"
    local terraformrc="${HOME}/.terraformrc"
    local dev_override_entry="\"${PROVIDER_SOURCE}\" = \"${install_dir}\""

    # Check if terraformrc exists and already has our override
    if [[ -f "$terraformrc" ]]; then
        if grep -q "${PROVIDER_SOURCE}" "$terraformrc"; then
            echo "  ~/.terraformrc already configured for ${PROVIDER_SOURCE}"
            return 0
        fi

        # Check if dev_overrides block exists
        if grep -q "dev_overrides" "$terraformrc"; then
            echo "  Warning: ~/.terraformrc has dev_overrides but not for ${PROVIDER_SOURCE}"
            echo "  Please add manually: ${dev_override_entry}"
            return 0
        fi
    fi

    # Create or append to terraformrc
    echo "  Configuring ~/.terraformrc..."

    if [[ -f "$terraformrc" ]]; then
        # Backup existing file
        cp "$terraformrc" "${terraformrc}.backup"
        echo "  Backed up existing ~/.terraformrc to ~/.terraformrc.backup"
    fi

    cat > "$terraformrc" << EOF
provider_installation {
  dev_overrides {
    "${PROVIDER_SOURCE}" = "${install_dir}"
  }
  direct {}
}
EOF
    echo "  Created ~/.terraformrc with dev_overrides"
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

    install_dir="${HOME}/.terraform.d/plugins/DO-Solutions/docidr"
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

    # Configure terraformrc
    configure_terraformrc "$install_dir"

    echo ""
    echo "Successfully installed terraform-provider-docidr v${version}"
    echo ""
    echo "Add the following to your Terraform configuration:"
    echo ""
    echo '  terraform {'
    echo '    required_providers {'
    echo '      docidr = {'
    echo "        source = \"${PROVIDER_SOURCE}\""
    echo '      }'
    echo '    }'
    echo '  }'
    echo ""
    echo "Note: You will see a warning about dev_overrides when running terraform."
    echo "This is expected behavior for providers installed from GitHub releases."
    echo ""
}

main "$@"
