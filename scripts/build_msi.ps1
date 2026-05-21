param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [string]$DistDir = "dist",
    [string]$BuildDirName = "mod-of-us_windows_x86_64",
    [string]$BinaryName = "Mod of Us.exe",
    [string]$OutputName = "mod-of-us_windows_x86_64.msi"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-MsiVersion {
    param([string]$InputVersion)

    $normalized = $InputVersion.Trim()
    if ($normalized.StartsWith("v")) {
        $normalized = $normalized.Substring(1)
    }
    $normalized = $normalized.Split("-", 2)[0]
    if ([string]::IsNullOrWhiteSpace($normalized)) {
        throw "Version is empty after normalization."
    }

    $parts = $normalized.Split(".")
    $numbers = @()
    foreach ($part in $parts) {
        if ($numbers.Count -ge 3) { break }
        if ($part -notmatch '^\d+$') {
            throw "Invalid version part: $part"
        }
        $numbers += [int]$part
    }
    while ($numbers.Count -lt 3) {
        $numbers += 0
    }

    return ($numbers -join ".")
}

$msiVersion = Get-MsiVersion -InputVersion $Version
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$distPath = Join-Path $repoRoot $DistDir
$buildPath = Join-Path $distPath $BuildDirName
$exePath = Join-Path $buildPath $BinaryName
$dllPath = Join-Path $repoRoot "lib\discord_partner_sdk.dll"
$iconPath = Join-Path $repoRoot "client\icon.ico"
$wxsPath = Join-Path $repoRoot "installer\wix\product.wxs"
$stagePath = Join-Path $distPath "wix"
$outputPath = Join-Path $distPath $OutputName

if (-not (Test-Path $exePath)) {
    $candidate = Get-ChildItem -Path $distPath -Directory -ErrorAction SilentlyContinue | Where-Object {
        Test-Path (Join-Path $_.FullName $BinaryName)
    } | Select-Object -First 1
    if ($candidate) {
        $buildPath = $candidate.FullName
        $exePath = Join-Path $buildPath $BinaryName
    }
}
if (-not (Test-Path $exePath)) {
    throw "Executable not found: $exePath"
}
if (-not (Test-Path $dllPath)) {
    throw "Discord SDK DLL not found: $dllPath"
}
if (-not (Test-Path $iconPath)) {
    throw "Icon not found: $iconPath"
}
if (-not (Test-Path $wxsPath)) {
    throw "WiX source not found: $wxsPath"
}

if (Test-Path $stagePath) {
    Remove-Item -Path $stagePath -Recurse -Force
}
New-Item -Path $stagePath -ItemType Directory | Out-Null

Copy-Item -Path $exePath -Destination $stagePath -Force
Copy-Item -Path $dllPath -Destination $stagePath -Force
Copy-Item -Path $iconPath -Destination $stagePath -Force

$wixCmd = Get-Command wix -ErrorAction SilentlyContinue
if (-not $wixCmd) {
    $dotnetCmd = Get-Command dotnet -ErrorAction SilentlyContinue
    if (-not $dotnetCmd) {
        throw "WiX not found and dotnet is unavailable. Install dotnet and run 'dotnet tool install --global wix --version 7.*'."
    }
    Write-Host "Installing WiX tool..."
    dotnet tool install --global wix --version 7.*
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to install WiX tool."
    }
    $env:PATH = "$env:USERPROFILE\\.dotnet\\tools;$env:PATH"
}

& wix eula accept wix7 | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Failed to accept WiX EULA (wix7)."
}

Write-Host "Ensuring WiX UI extension is installed..."
& wix extension add WixToolset.UI.wixext
if ($LASTEXITCODE -ne 0) {
    throw "Failed to install WiX UI extension."
}

$wixArgs = @(
    "build",
    "-nologo",
    "-acceptEula", "wix7",
    "-ext", "WixToolset.UI.wixext",
    "-d", "SourceDir=$stagePath",
    "-d", "ProductVersion=$msiVersion",
    "-out", $outputPath,
    $wxsPath
)

Write-Host "Building MSI version $msiVersion"
& wix @wixArgs
if ($LASTEXITCODE -ne 0) {
    throw "wix build failed with exit code $LASTEXITCODE"
}

Write-Host "MSI generated: $outputPath"
