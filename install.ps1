#Requires -Version 5.1
<#
.SYNOPSIS
    Install script for DFIR Lab CLI (dfir-cli).

.DESCRIPTION
    Downloads and installs the latest (or a specified) version of dfir-cli
    from GitHub Releases. Verifies SHA256 checksums and adds the install
    directory to the user PATH.

.PARAMETER Version
    Specific version to install (e.g. "0.3.1"). Defaults to the latest release.

.PARAMETER InstallDir
    Custom installation directory. Defaults to $env:USERPROFILE\.dfir-cli\bin

.EXAMPLE
    iwr https://raw.githubusercontent.com/ForeGuards/dfir-cli/main/install.ps1 | iex

.EXAMPLE
    .\install.ps1 -Version 0.3.1

.EXAMPLE
    .\install.ps1 -InstallDir "C:\Tools\dfir-cli"
#>

[CmdletBinding()]
param(
    [string]$Version,
    [string]$InstallDir
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
$GH_REPO       = "ForeGuards/dfir-cli-releases"
$GH_API_LATEST = "https://api.github.com/repos/$GH_REPO/releases/latest"
$BINARY_NAME   = "dfir-cli.exe"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
function Write-Status  { param([string]$Message) Write-Host "  [*] " -ForegroundColor Cyan  -NoNewline; Write-Host $Message }
function Write-Ok      { param([string]$Message) Write-Host "  [+] " -ForegroundColor Green -NoNewline; Write-Host $Message }
function Write-Warn    { param([string]$Message) Write-Host "  [!] " -ForegroundColor Yellow -NoNewline; Write-Host $Message }
function Write-Err     { param([string]$Message) Write-Host "  [-] " -ForegroundColor Red   -NoNewline; Write-Host $Message }

function Exit-WithError {
    param([string]$Message)
    Write-Err $Message
    Write-Error $Message
    exit 1
}

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------
function Show-Banner {
    Write-Host ""
    Write-Host "  ========================================" -ForegroundColor DarkCyan
    Write-Host "       DFIR Lab CLI Installer"              -ForegroundColor White
    Write-Host "  ========================================" -ForegroundColor DarkCyan
    Write-Host ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
function Install-DfirCli {
    Show-Banner

    # --- TLS 1.2+ ----------------------------------------------------------
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls13
    } catch {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    }

    # --- Architecture check -------------------------------------------------
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ($arch -ne "AMD64") {
        Exit-WithError "Unsupported architecture: $arch. Only x86_64 (AMD64) is supported at this time."
    }
    Write-Status "Architecture: $arch"

    # --- Resolve version ----------------------------------------------------
    if (-not $Version) {
        Write-Status "Fetching latest release from GitHub..."
        try {
            $release = Invoke-RestMethod -Uri $GH_API_LATEST -Headers @{ "User-Agent" = "dfir-cli-installer" }
            $Version = $release.tag_name -replace '^v', ''
        }
        catch {
            Exit-WithError "Failed to fetch latest release: $_"
        }
    }
    $Version = $Version -replace '^v', ''
    if ($Version -notmatch '^\d+\.\d+\.\d+(-[\w.]+)?$') {
        Exit-WithError "Invalid version format: $Version. Expected format: 1.2.3"
    }
    Write-Ok "Version: $Version"

    # --- Resolve install directory -------------------------------------------
    if (-not $InstallDir) {
        $InstallDir = Join-Path $env:USERPROFILE ".dfir-cli\bin"
    }
    Write-Status "Install directory: $InstallDir"

    # --- Prepare temp directory ----------------------------------------------
    $tempDir = Join-Path ([System.IO.Path]::GetTempPath()) "dfir-cli-install-$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    Write-Status "Temp directory: $tempDir"

    try {
        # --- Build URLs ------------------------------------------------------
        $zipName       = "dfir-cli_${Version}_windows_amd64.zip"
        $checksumsName = "dfir-cli_${Version}_checksums.txt"
        $baseUrl       = "https://github.com/$GH_REPO/releases/download/v${Version}"
        $zipUrl        = "$baseUrl/$zipName"
        $checksumsUrl  = "$baseUrl/$checksumsName"

        $zipPath       = Join-Path $tempDir $zipName
        $checksumsPath = Join-Path $tempDir $checksumsName

        # --- Download assets --------------------------------------------------
        # Validate HTTPS URLs before downloading
        if ($zipUrl -notmatch '^https://') {
            Exit-WithError "Refusing non-HTTPS download URL: $zipUrl"
        }
        if ($checksumsUrl -notmatch '^https://') {
            Exit-WithError "Refusing non-HTTPS download URL: $checksumsUrl"
        }

        Write-Status "Downloading $zipName..."
        try {
            Invoke-WebRequest -Uri $zipUrl -OutFile $zipPath -UseBasicParsing
        }
        catch {
            Exit-WithError "Failed to download $zipUrl`n$_"
        }
        Write-Ok "Downloaded zip ($('{0:N2}' -f ((Get-Item $zipPath).Length / 1MB)) MB)"

        Write-Status "Downloading $checksumsName..."
        try {
            Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath -UseBasicParsing
        }
        catch {
            Exit-WithError "Failed to download $checksumsUrl`n$_"
        }
        Write-Ok "Downloaded checksums file"

        # --- Verify SHA256 checksum -------------------------------------------
        Write-Status "Verifying SHA256 checksum..."
        $expectedLine = Get-Content $checksumsPath | Where-Object { $_ -match [regex]::Escape($zipName) }
        if (-not $expectedLine) {
            Exit-WithError "Checksum entry for $zipName not found in $checksumsName"
        }
        $expectedHash = ($expectedLine -split '\s+')[0].Trim().ToUpper()

        $actualHash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToUpper()
        if ($actualHash -ne $expectedHash) {
            Exit-WithError "Checksum mismatch!`n  Expected: $expectedHash`n  Actual:   $actualHash"
        }
        Write-Ok "Checksum verified: $actualHash"

        # --- Extract binary ---------------------------------------------------
        Write-Status "Extracting $BINARY_NAME..."
        $extractDir = Join-Path $tempDir "extract"
        Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force

        $binarySource = Get-ChildItem -Path $extractDir -Recurse -Filter $BINARY_NAME | Select-Object -First 1
        if (-not $binarySource) {
            Exit-WithError "$BINARY_NAME not found inside the zip archive."
        }

        # --- Install binary ---------------------------------------------------
        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
            Write-Status "Created directory: $InstallDir"
        }

        $binaryDest = Join-Path $InstallDir $BINARY_NAME
        Copy-Item -Path $binarySource.FullName -Destination $binaryDest -Force
        Write-Ok "Installed $BINARY_NAME to $binaryDest"

        # --- Update PATH ------------------------------------------------------
        $pathModified = $false
        $userPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
        $pathEntries = $userPath -split ';' | Where-Object { $_.Trim() -ne '' }

        if ($pathEntries -notcontains $InstallDir) {
            Write-Status "Adding $InstallDir to user PATH..."
            $newPath = ($pathEntries + $InstallDir) -join ';'
            [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
            Write-Ok "Updated user PATH"
            $pathModified = $true
        }
        else {
            Write-Ok "Install directory already in PATH"
        }

        # --- Done -------------------------------------------------------------
        Write-Host ""
        Write-Host "  ========================================" -ForegroundColor DarkCyan
        Write-Host "       Installation complete!"              -ForegroundColor Green
        Write-Host "  ========================================" -ForegroundColor DarkCyan
        Write-Host ""
        Write-Ok "dfir-cli v$Version installed to $InstallDir"
        Write-Host ""

        if ($pathModified) {
            Write-Warn "Your PATH has been updated. Please restart your terminal (or open a new one) for the changes to take effect."
            Write-Host ""
        }

        Write-Status "Run 'dfir-cli --help' to get started."
        Write-Host ""
    }
    finally {
        # --- Cleanup ----------------------------------------------------------
        if (Test-Path $tempDir) {
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------
Install-DfirCli
