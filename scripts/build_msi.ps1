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

$wixArgs = @(
    "build",
    "-nologo",
    "-ext", "WixToolset.UI.wixext",
    "-dSourceDir=$stagePath",
    "-dProductVersion=$msiVersion",
    "-out", $outputPath,
    $wxsPath
)

Write-Host "Building MSI version $msiVersion"
& wix @wixArgs
if ($LASTEXITCODE -ne 0) {
    throw "wix build failed with exit code $LASTEXITCODE"
}

Write-Host "MSI generated: $outputPath"
