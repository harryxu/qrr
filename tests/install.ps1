# Offline integration tests; no GitHub requests or administrator access required.
$ErrorActionPreference = 'Stop'
$repositoryDir = Split-Path -Parent $PSScriptRoot
$workspace = Join-Path ([IO.Path]::GetTempPath()) ('qrr-install-test-' + [guid]::NewGuid().ToString('N'))
$originalUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$originalEnvironment = @{}
foreach ($name in @('OS', 'PROCESSOR_ARCHITECTURE', 'PROCESSOR_ARCHITEW6432', 'LOCALAPPDATA', 'QRR_INSTALL_DIR', 'Path')) {
    $originalEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

function Invoke-RestMethod {
    param($Uri, $Headers, $TimeoutSec)
    if ($Uri -ne 'https://api.github.com/repos/harryxu/qrr/releases/latest') {
        throw "Unexpected request: $Uri"
    }
    return $global:QrrInstallerTestRelease
}

function Invoke-WebRequest {
    param([switch]$UseBasicParsing, $Uri, $OutFile, $TimeoutSec)
    if ($global:QrrInstallerTestDownloadFailure) { throw 'Fixture download failure' }
    switch ($Uri) {
        'https://example.invalid/archive' { Copy-Item -LiteralPath $global:QrrInstallerTestArchive -Destination $OutFile }
        'https://example.invalid/checksums' { Set-Content -LiteralPath $OutFile -Value $global:QrrInstallerTestChecksums -Encoding ASCII }
        default { throw "Unexpected download: $Uri" }
    }
}

function Assert-Installed {
    $binary = Join-Path $env:QRR_INSTALL_DIR 'qrr.exe'
    if ((Get-Content -LiteralPath $binary -Raw).Trim() -ne 'qrr installer fixture') {
        throw 'Installed binary changed unexpectedly.'
    }
}

function Assert-Failure {
    param([string]$Message)
    $failure = $null
    try { & (Join-Path $repositoryDir 'install.ps1') }
    catch { $failure = $_.Exception.Message }
    if (-not $failure -or $failure -notlike "*$Message*") {
        throw "Expected failure containing '$Message', got '$failure'."
    }
    Assert-Installed
}

try {
    $null = New-Item -ItemType Directory -Path $workspace
    $fixtureBinary = Join-Path $workspace 'qrr.exe'
    Set-Content -LiteralPath $fixtureBinary -Value 'qrr installer fixture' -Encoding ASCII
    $global:QrrInstallerTestArchive = Join-Path $workspace 'archive.zip'
    Compress-Archive -LiteralPath $fixtureBinary -DestinationPath $global:QrrInstallerTestArchive
    $archiveName = 'qrr_v1.2.3_windows_amd64.zip'
    $checksum = (Get-FileHash -LiteralPath $global:QrrInstallerTestArchive -Algorithm SHA256).Hash
    $global:QrrInstallerTestChecksums = "$checksum  ./$archiveName"
    $global:QrrInstallerTestDownloadFailure = $false
    $global:QrrInstallerTestRelease = [pscustomobject]@{
        tag_name = 'v1.2.3'; draft = $false; prerelease = $false
        assets = @(
            [pscustomobject]@{ name = $archiveName; browser_download_url = 'https://example.invalid/archive' },
            [pscustomobject]@{ name = 'checksums.txt'; browser_download_url = 'https://example.invalid/checksums' }
        )
    }
    $env:OS = 'Windows_NT'
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    $env:PROCESSOR_ARCHITEW6432 = $null
    $env:QRR_INSTALL_DIR = Join-Path $workspace 'install directory'

    Get-Content -LiteralPath (Join-Path $repositoryDir 'install.ps1') -Raw | Invoke-Expression
    Assert-Installed
    & (Join-Path $repositoryDir 'install.ps1')
    Assert-Installed
    $processEntries = @($env:Path -split ';' | Where-Object { $_ -eq $env:QRR_INSTALL_DIR })
    if ($processEntries.Count -ne 1) { throw 'Process PATH contains duplicate installation entries.' }
    if ([Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT) {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        $userEntries = @($userPath -split ';' | Where-Object { $_ -eq $env:QRR_INSTALL_DIR })
        if ($userEntries.Count -ne 1) { throw 'User PATH must contain the install directory exactly once.' }
        foreach ($entry in @($originalUserPath -split ';' | Where-Object { $_ })) {
            if (($userPath -split ';') -notcontains $entry) { throw 'An existing user PATH entry was removed.' }
        }
    }
    Write-Host 'PASS: checksum verification, paths with spaces, upgrades, and PATH preservation'

    $customDir = $env:QRR_INSTALL_DIR
    $env:LOCALAPPDATA = Join-Path $workspace 'local app data'
    $env:QRR_INSTALL_DIR = $null
    & (Join-Path $repositoryDir 'install.ps1')
    $env:QRR_INSTALL_DIR = Join-Path $env:LOCALAPPDATA 'Programs\qrr\bin'
    Assert-Installed
    $env:QRR_INSTALL_DIR = $customDir
    Write-Host 'PASS: default user installation directory'

    # Detect the native OS architecture when running a 32-bit PowerShell process.
    $env:PROCESSOR_ARCHITECTURE = 'x86'
    $env:PROCESSOR_ARCHITEW6432 = 'AMD64'
    & (Join-Path $repositoryDir 'install.ps1')
    Assert-Installed
    $env:PROCESSOR_ARCHITEW6432 = 'ARM64'
    Assert-Failure 'Windows amd64 is required'
    $env:PROCESSOR_ARCHITEW6432 = $null
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    $env:OS = 'Linux'
    Assert-Failure 'This installer supports Windows'
    $env:OS = 'Windows_NT'
    Write-Host 'PASS: native architecture detection and unsupported platforms'

    $global:QrrInstallerTestDownloadFailure = $true
    Assert-Failure 'Fixture download failure'
    $global:QrrInstallerTestDownloadFailure = $false
    $global:QrrInstallerTestChecksums = ('0' * 64) + "  ./$archiveName"
    Assert-Failure 'failed SHA-256 verification'
    $global:QrrInstallerTestChecksums = "invalid  ./$archiveName"
    Assert-Failure 'Missing, invalid, or duplicate checksum'
    $global:QrrInstallerTestChecksums = ''
    Assert-Failure 'Missing, invalid, or duplicate checksum'
    $global:QrrInstallerTestChecksums = "$checksum  ./$archiveName`n$checksum  ./$archiveName"
    Assert-Failure 'Missing, invalid, or duplicate checksum'
    $global:QrrInstallerTestRelease.assets = @()
    Assert-Failure 'must contain'
    $global:QrrInstallerTestRelease.prerelease = $true
    Assert-Failure 'stable published release'
    Write-Host 'PASS: download errors, missing assets, prereleases, and invalid checksums preserve the installed binary'
}
finally {
    Remove-Variable -Name QrrInstallerTest* -Scope Global -ErrorAction SilentlyContinue
    [Environment]::SetEnvironmentVariable('Path', $originalUserPath, 'User')
    foreach ($name in $originalEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $originalEnvironment[$name], 'Process')
    }
    if (Test-Path -LiteralPath $workspace) {
        Remove-Item -LiteralPath $workspace -Recurse -Force
    }
}
