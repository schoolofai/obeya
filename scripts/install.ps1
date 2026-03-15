# Obeya (ob) installer for Windows
# Usage:
#   irm https://raw.githubusercontent.com/schoolofai/obeya/main/scripts/install.ps1 | iex
#   ./install.ps1 -Version 0.2.0

[CmdletBinding()]
param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

$Repo = "schoolofai/obeya"
$Binary = "ob.exe"
$InstallDir = Join-Path $env:LOCALAPPDATA "obeya\bin"

# Detect architecture
$Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    "X64"   { "amd64" }
    "Arm64" { "arm64" }
    default {
        Write-Error "Unsupported architecture: $_"
        exit 1
    }
}

# Windows arm64 is not built — fall back notice
if ($Arch -eq "arm64") {
    Write-Error "Windows ARM64 builds are not available yet. Use the amd64 build under emulation."
    exit 1
}

# Get latest version from GitHub API if not specified
if (-not $Version) {
    Write-Host "Fetching latest version..."
    try {
        $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        $Version = $Release.tag_name -replace '^v', ''
    } catch {
        Write-Error "Could not determine latest version. Specify one with -Version."
        exit 1
    }
}

$Archive = "obeya_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/v$Version/$Archive"

Write-Host "Installing obeya v$Version (windows/$Arch)..."

# Create temp directory
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    # Download
    Write-Host "Downloading $Url..."
    $ArchivePath = Join-Path $TmpDir $Archive
    try {
        Invoke-WebRequest -Uri $Url -OutFile $ArchivePath -UseBasicParsing
    } catch {
        Write-Error "Download failed. Check that version v$Version exists at: https://github.com/$Repo/releases"
        exit 1
    }

    # Extract
    Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force

    # Install binary
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    Copy-Item -Path (Join-Path $TmpDir $Binary) -Destination (Join-Path $InstallDir $Binary) -Force

    Write-Host ""
    Write-Host "Successfully installed ob to $InstallDir\$Binary"

    # Add to user PATH if not already present
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
        Write-Host ""
        Write-Host "Added $InstallDir to your user PATH."
        Write-Host "Restart your terminal, then run 'ob --help' to get started."
    } else {
        Write-Host "Run 'ob --help' to get started."
    }
} finally {
    # Clean up
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
