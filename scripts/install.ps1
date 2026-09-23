# upp Windows installer — downloads the matching release archive and verifies its
# SHA-256 checksum against checksums.txt before installing.
# Usage: irm https://raw.githubusercontent.com/JhnFrankz/upp/main/scripts/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$repo = "JhnFrankz/upp"
$binary = "upp.exe"

# Architecture detection
$rawArch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
$arch = switch -Regex ($rawArch) {
    '^(AMD64|x64)$'    { 'amd64' }
    '^ARM64$'         { 'arm64' }
    '^(x86|i[3-6]86)$' {
        Write-Error "upp does not support 32-bit Windows ($rawArch). Only 64-bit (x64 / arm64) architectures are supported."
        exit 1
    }
    Default {
        Write-Error "Unsupported architecture: $rawArch. Only amd64 and arm64 are supported."
        exit 1
    }
}

Write-Host "Detected architecture: $arch"

# Resolve version
if ($env:VERSION -and $env:VERSION -ne '' -and $env:VERSION -ne 'latest') {
    $tag = $env:VERSION
    Write-Host "Using specified version: $tag"
} else {
    Write-Host "Querying latest release for $repo..."
    $apiUri = "https://api.github.com/repos/$repo/releases/latest"
    try {
        $headers = @{ "User-Agent" = "upp-installer" }
        $release = Invoke-RestMethod -Uri $apiUri -Headers $headers -UseBasicParsing
        $tag = $release.tag_name
    } catch {
        Write-Error "Failed to fetch latest release metadata from GitHub API: $_"
        exit 1
    }
}

if (-not $tag) {
    Write-Error "Could not determine the release tag."
    exit 1
}

Write-Host "Resolved release tag: $tag"

# URLs
$assetName = "upp-windows-$arch.zip"
$assetUrl = "https://github.com/$repo/releases/download/$tag/$assetName"
$checksumsUrl = "https://github.com/$repo/releases/download/$tag/checksums.txt"

# Temporary directory
$tempDir = Join-Path $env:TEMP ("upp-install-" + [System.Guid]::NewGuid().ToString("n"))
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null

try {
    $zipPath = Join-Path $tempDir $assetName
    $checksumsPath = Join-Path $tempDir "checksums.txt"

    Write-Host "Downloading $assetName..."
    Invoke-WebRequest -Uri $assetUrl -OutFile $zipPath -UseBasicParsing

    Write-Host "Downloading checksums.txt..."
    Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath -UseBasicParsing

    # Checksum verification
    Write-Host "Verifying SHA-256 checksum..."
    $actualHash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLower()

    $expectedHash = $null
    $checksumLines = Get-Content -Path $checksumsPath
    foreach ($line in $checksumLines) {
        $trimmed = $line.Trim()
        if ([string]::IsNullOrWhiteSpace($trimmed)) {
            continue
        }
        $parts = -split $trimmed
        if ($parts.Length -eq 2) {
            $sum = $parts[0]
            $file = $parts[1].TrimStart('*')
            if ($file -eq $assetName) {
                $expectedHash = $sum.ToLower()
                break
            }
        }
    }

    if (-not $expectedHash) {
        Write-Error "Checksum for $assetName not found in checksums.txt."
        exit 1
    }

    if ($actualHash -ne $expectedHash) {
        Write-Error "Checksum verification failed!`nExpected: $expectedHash`nActual:   $actualHash"
        exit 1
    }
    Write-Host "Checksum verified: $actualHash"

    # Extraction
    $extractDir = Join-Path $tempDir "extract"
    Write-Host "Extracting archive..."
    Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force

    $extractedBin = Join-Path $extractDir (Join-Path "upp-windows-$arch" $binary)
    if (-not (Test-Path $extractedBin)) {
        $found = Get-ChildItem -Path $extractDir -Filter $binary -Recurse | Select-Object -First 1
        if ($found) {
            $extractedBin = $found.FullName
        } else {
            Write-Error "$binary not found in extracted archive."
            exit 1
        }
    }

    # Install location
    $installDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "upp\bin" }
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    }

    $targetExe = Join-Path $installDir $binary
    Write-Host "Installing $binary to $targetExe..."
    Copy-Item -Path $extractedBin -Destination $targetExe -Force

    # PATH configuration
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $pathEntries = if ($userPath) { $userPath -split ';' } else { @() }
    $inPath = $false
    foreach ($p in $pathEntries) {
        if ($p.Trim().TrimEnd('\') -ieq $installDir.Trim().TrimEnd('\')) {
            $inPath = $true
            break
        }
    }

    if (-not $inPath) {
        Write-Host "Adding $installDir to User PATH..."
        $newUserPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $installDir } else { "$userPath;$installDir" }
        [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
        $env:PATH = "$installDir;$env:PATH"
    }

    # Smoke check
    Write-Host "Running smoke check..."
    & "$targetExe" --version

    Write-Host "`nupp has been installed successfully to $targetExe!"
    Write-Host "Run 'upp --help' to get started."
} finally {
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force -Path $tempDir -ErrorAction SilentlyContinue
    }
}
