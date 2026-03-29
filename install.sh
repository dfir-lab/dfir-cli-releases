#!/bin/sh
# =============================================================================
# DFIR Lab CLI Installer
# =============================================================================
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/dfir-lab/dfir-cli-releases/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/dfir-lab/dfir-cli-releases/main/install.sh | sh -s -- --version 1.2.0
#   curl -fsSL https://raw.githubusercontent.com/dfir-lab/dfir-cli-releases/main/install.sh | sh -s -- --install-dir /opt/bin
#
# This script downloads and installs the dfir-cli binary for the current
# platform. It verifies the download using SHA256 checksums before installing.
#
# Supported platforms:
#   - macOS (Darwin) amd64/arm64
#   - Linux amd64/arm64
#
# =============================================================================

set -e

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

GITHUB_REPO="dfir-lab/dfir-cli-releases"
GITHUB_API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
GITHUB_DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download"
BINARY_NAME="dfir-cli"

# ---------------------------------------------------------------------------
# Color helpers (only when connected to a terminal)
# ---------------------------------------------------------------------------

if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    RESET='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    RESET=''
fi

info()    { printf "${BLUE}[*]${RESET} %s\n" "$1"; }
success() { printf "${GREEN}[+]${RESET} %s\n" "$1"; }
warn()    { printf "${YELLOW}[!]${RESET} %s\n" "$1"; }
error()   { printf "${RED}[-]${RESET} %s\n" "$1" >&2; }
fatal()   { error "$1"; exit 1; }

# ---------------------------------------------------------------------------
# Cleanup trap — always remove the temp directory on exit
# ---------------------------------------------------------------------------

TMPDIR_INSTALL=""

cleanup() {
    if [ -n "${TMPDIR_INSTALL}" ] && [ -d "${TMPDIR_INSTALL}" ]; then
        rm -rf "${TMPDIR_INSTALL}"
    fi
}

trap cleanup EXIT INT TERM

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------

REQUESTED_VERSION=""
INSTALL_DIR=""

while [ $# -gt 0 ]; do
    case "$1" in
        --version)
            shift
            [ $# -eq 0 ] && fatal "Missing value for --version"
            REQUESTED_VERSION="$1"
            # Validate version format: optional leading v, then digits and dots
            case "${REQUESTED_VERSION}" in
                v[0-9]*|[0-9]*) ;; # looks valid
                *) fatal "Invalid version format: ${REQUESTED_VERSION}. Expected: 1.2.3 or v1.2.3" ;;
            esac
            ;;
        --install-dir)
            shift
            [ $# -eq 0 ] && fatal "Missing value for --install-dir"
            INSTALL_DIR="$1"
            ;;
        --help|-h)
            printf "Usage: install.sh [--version VERSION] [--install-dir DIR]\n"
            exit 0
            ;;
        *)
            fatal "Unknown option: $1"
            ;;
    esac
    shift
done

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------

printf "\n"
printf "${BOLD}  ____  _____ ___ ____    _          _        ____ _     ___${RESET}\n"
printf "${BOLD} |  _ \\|  ___|_ _|  _ \\  | |    __ _| |__    / ___| |   |_ _|${RESET}\n"
printf "${BOLD} | | | | |_   | || |_) | | |   / _\` | '_ \\  | |   | |    | |${RESET}\n"
printf "${BOLD} | |_| |  _|  | ||  _ <  | |__| (_| | |_) | | |___| |___ | |${RESET}\n"
printf "${BOLD} |____/|_|   |___|_| \\_\\ |_____\\__,_|_.__/   \\____|_____|___|${RESET}\n"
printf "\n"
printf "  ${BLUE}DFIR Lab CLI Installer${RESET}\n"
printf "\n"

# ---------------------------------------------------------------------------
# Detect HTTP client (prefer curl, fall back to wget)
# ---------------------------------------------------------------------------

DOWNLOADER=""
DOWNLOADER_QUIET=""

detect_downloader() {
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
    else
        fatal "Neither 'curl' nor 'wget' found. Please install one and try again."
    fi
    info "Using ${DOWNLOADER} for downloads"
}

# Download a URL to a local file.
# Usage: download_file <url> <output_path>
download_file() {
    _url="$1"
    _output="$2"

    if [ "${DOWNLOADER}" = "curl" ]; then
        curl -fsSL --proto '=https' --tlsv1.2 -o "${_output}" "${_url}"
    else
        wget -q --https-only -O "${_output}" "${_url}"
    fi
}

# Fetch a URL and print its contents to stdout.
# Usage: download_stdout <url>
download_stdout() {
    _url="$1"

    if [ "${DOWNLOADER}" = "curl" ]; then
        curl -fsSL --proto '=https' --tlsv1.2 "${_url}"
    else
        wget -q --https-only -O - "${_url}"
    fi
}

# ---------------------------------------------------------------------------
# Detect platform (OS and architecture)
# ---------------------------------------------------------------------------

detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "${OS}" in
        Darwin) OS="darwin" ;;
        Linux)  OS="linux"  ;;
        *)      fatal "Unsupported operating system: ${OS}" ;;
    esac

    case "${ARCH}" in
        x86_64|amd64)       ARCH="amd64" ;;
        aarch64|arm64)       ARCH="arm64" ;;
        *)                   fatal "Unsupported architecture: ${ARCH}" ;;
    esac

    info "Detected platform: ${OS}/${ARCH}"
}

# ---------------------------------------------------------------------------
# Resolve the version to install
# ---------------------------------------------------------------------------

resolve_version() {
    if [ -n "${REQUESTED_VERSION}" ]; then
        # Strip leading "v" if the user included it
        VERSION="$(printf '%s' "${REQUESTED_VERSION}" | sed 's/^v//')"
        info "Using requested version: ${VERSION}"
    else
        info "Fetching latest release from GitHub..."

        # The GitHub API returns JSON; extract the tag_name field without
        # requiring jq by using a simple sed/grep pipeline.
        RELEASE_JSON="$(download_stdout "${GITHUB_API_URL}")" \
            || fatal "Failed to fetch latest release information from GitHub"

        TAG="$(printf '%s' "${RELEASE_JSON}" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"

        [ -z "${TAG}" ] && fatal "Could not determine latest release version"

        # Strip leading "v" from the tag (e.g. v1.0.0 -> 1.0.0)
        VERSION="$(printf '%s' "${TAG}" | sed 's/^v//')"
        info "Latest version: ${VERSION}"
    fi
}

# ---------------------------------------------------------------------------
# Download and verify the release
# ---------------------------------------------------------------------------

download_and_verify() {
    TARBALL_NAME="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
    CHECKSUMS_NAME="${BINARY_NAME}_${VERSION}_checksums.txt"

    TARBALL_URL="${GITHUB_DOWNLOAD_URL}/v${VERSION}/${TARBALL_NAME}"
    CHECKSUMS_URL="${GITHUB_DOWNLOAD_URL}/v${VERSION}/${CHECKSUMS_NAME}"

    # Create a secure temporary directory
    TMPDIR_INSTALL="$(mktemp -d)" \
        || fatal "Failed to create temporary directory"

    info "Downloading ${TARBALL_NAME}..."
    download_file "${TARBALL_URL}" "${TMPDIR_INSTALL}/${TARBALL_NAME}" \
        || fatal "Failed to download tarball from ${TARBALL_URL}"

    info "Downloading checksums..."
    download_file "${CHECKSUMS_URL}" "${TMPDIR_INSTALL}/${CHECKSUMS_NAME}" \
        || fatal "Failed to download checksums from ${CHECKSUMS_URL}"

    # -----------------------------------------------------------------------
    # Verify SHA256 checksum
    # -----------------------------------------------------------------------

    # NOTE: Checksum verification protects against download corruption and
    # CDN/mirror tampering, but not against a compromised GitHub release.
    # For stronger guarantees, verify GPG/cosign signatures (see docs).
    info "Verifying SHA256 checksum..."

    # Extract the expected checksum for our tarball from the checksums file
    EXPECTED_CHECKSUM="$(grep -F "${TARBALL_NAME}" "${TMPDIR_INSTALL}/${CHECKSUMS_NAME}" | awk '{print $1}')"

    [ -z "${EXPECTED_CHECKSUM}" ] && fatal "Checksum not found for ${TARBALL_NAME} in checksums file"

    # Compute the actual checksum
    if [ "${OS}" = "darwin" ]; then
        ACTUAL_CHECKSUM="$(shasum -a 256 "${TMPDIR_INSTALL}/${TARBALL_NAME}" | awk '{print $1}')"
    else
        ACTUAL_CHECKSUM="$(sha256sum "${TMPDIR_INSTALL}/${TARBALL_NAME}" | awk '{print $1}')"
    fi

    if [ "${EXPECTED_CHECKSUM}" != "${ACTUAL_CHECKSUM}" ]; then
        error "Checksum verification failed!"
        error "  Expected: ${EXPECTED_CHECKSUM}"
        error "  Actual:   ${ACTUAL_CHECKSUM}"
        fatal "The downloaded file may be corrupted or tampered with. Aborting."
    fi

    success "Checksum verified"
}

# ---------------------------------------------------------------------------
# Extract and install the binary
# ---------------------------------------------------------------------------

install_binary() {
    info "Extracting binary..."
    tar -xzf "${TMPDIR_INSTALL}/${TARBALL_NAME}" -C "${TMPDIR_INSTALL}" \
        || fatal "Failed to extract tarball"

    # Ensure the binary exists after extraction
    [ -f "${TMPDIR_INSTALL}/${BINARY_NAME}" ] \
        || fatal "Binary '${BINARY_NAME}' not found in tarball"

    chmod +x "${TMPDIR_INSTALL}/${BINARY_NAME}"

    # -----------------------------------------------------------------------
    # Determine install directory
    # -----------------------------------------------------------------------

    NEEDS_PATH_WARNING=false

    if [ -n "${INSTALL_DIR}" ]; then
        # User specified a custom install directory
        TARGET_DIR="${INSTALL_DIR}"
    elif [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
        TARGET_DIR="/usr/local/bin"
    elif command -v sudo >/dev/null 2>&1; then
        # Try with sudo
        TARGET_DIR="/usr/local/bin"
    else
        # Fall back to user-local directory
        TARGET_DIR="${HOME}/.local/bin"
        NEEDS_PATH_WARNING=true
    fi

    # Create the target directory if it does not exist
    if [ ! -d "${TARGET_DIR}" ]; then
        if [ "${TARGET_DIR}" = "/usr/local/bin" ]; then
            sudo mkdir -p "${TARGET_DIR}" \
                || fatal "Failed to create ${TARGET_DIR}"
        else
            mkdir -p "${TARGET_DIR}" \
                || fatal "Failed to create ${TARGET_DIR}"
        fi
    fi

    # -----------------------------------------------------------------------
    # Copy the binary into place
    # -----------------------------------------------------------------------

    info "Installing ${BINARY_NAME} to ${TARGET_DIR}..."

    if [ -w "${TARGET_DIR}" ]; then
        cp "${TMPDIR_INSTALL}/${BINARY_NAME}" "${TARGET_DIR}/${BINARY_NAME}" \
            || fatal "Failed to install binary to ${TARGET_DIR}"
    else
        info "Elevated permissions required — running with sudo"
        sudo cp "${TMPDIR_INSTALL}/${BINARY_NAME}" "${TARGET_DIR}/${BINARY_NAME}" \
            || fatal "Failed to install binary to ${TARGET_DIR} (even with sudo)"
    fi

    success "Installed ${BINARY_NAME} to ${TARGET_DIR}/${BINARY_NAME}"

    # Verify the installed binary runs correctly
    INSTALLED_VERSION="$("${TARGET_DIR}/${BINARY_NAME}" --version 2>/dev/null || true)"
    if [ -n "${INSTALLED_VERSION}" ]; then
        info "Verified: ${INSTALLED_VERSION}"
    fi

    # Check if the install dir is on PATH; warn if not
    case ":${PATH}:" in
        *":${TARGET_DIR}:"*) ;;
        *)
            NEEDS_PATH_WARNING=true
            ;;
    esac

    if [ "${NEEDS_PATH_WARNING}" = true ]; then
        printf "\n"
        warn "${TARGET_DIR} is not in your PATH."
        warn "Add it by appending one of the following to your shell profile:"
        printf "\n"
        printf "    ${BOLD}export PATH=\"%s:\$PATH\"${RESET}\n" "${TARGET_DIR}"
        printf "\n"
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
    detect_downloader
    detect_platform
    resolve_version
    download_and_verify
    install_binary

    printf "\n"
    success "${BOLD}${BINARY_NAME} v${VERSION}${RESET}${GREEN} installed successfully!${RESET}"
    printf "\n"
    printf "  Run ${BOLD}${BINARY_NAME} --help${RESET} to get started.\n"
    printf "\n"
}

main
