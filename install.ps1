# Run in a script block so downloading with Invoke-Expression does not change
# the caller's error preferences or leave installer variables in their session.
& {
    $ErrorActionPreference = 'Stop'
    $temporaryDir = $null
    $stagedBinary = $null

    try {
        if ($env:OS -ne 'Windows_NT') {
            throw 'This installer supports Windows. Use install.sh on macOS or Linux.'
        }
        $architecture = $env:PROCESSOR_ARCHITEW6432
        if (-not $architecture) {
            $architecture = $env:PROCESSOR_ARCHITECTURE
        }
        if ($architecture -ne 'AMD64') {
            throw "No Windows release binary is available for architecture: $architecture. Windows amd64 is required."
        }

        $installDir = $env:QRR_INSTALL_DIR
        if (-not $installDir) {
            if (-not $env:LOCALAPPDATA) {
                throw 'LOCALAPPDATA is not set. Set QRR_INSTALL_DIR to the destination directory.'
            }
            $installDir = Join-Path $env:LOCALAPPDATA 'Programs\qrr\bin'
        }
        $installDir = [System.IO.Path]::GetFullPath($installDir)

        Write-Host 'Detecting the latest qrr release for windows/amd64...'
        $headers = @{ Accept = 'application/vnd.github+json'; 'User-Agent' = 'qrr-installer' }
        $release = Invoke-RestMethod -Uri 'https://api.github.com/repos/harryxu/qrr/releases/latest' -Headers $headers -TimeoutSec 30
        if ($release.draft -or $release.prerelease -or -not $release.tag_name) {
            throw 'GitHub did not return a stable published release.'
        }
        $version = $release.tag_name -replace '[^a-zA-Z0-9._-]', '_'
        $archiveName = "qrr_${version}_windows_amd64.zip"
        $archiveAsset = @($release.assets | Where-Object { $_.name -eq $archiveName })
        $checksumAsset = @($release.assets | Where-Object { $_.name -eq 'checksums.txt' })
        if ($archiveAsset.Count -ne 1 -or $checksumAsset.Count -ne 1) {
            throw "Release $($release.tag_name) must contain $archiveName and checksums.txt."
        }

        $temporaryDir = Join-Path ([System.IO.Path]::GetTempPath()) ('qrr-install-' + [guid]::NewGuid().ToString('N'))
        $null = New-Item -ItemType Directory -Path $temporaryDir
        $archivePath = Join-Path $temporaryDir $archiveName
        $checksumPath = Join-Path $temporaryDir 'checksums.txt'
        Write-Host "Downloading qrr $($release.tag_name)..."
        Invoke-WebRequest -UseBasicParsing -Uri $archiveAsset[0].browser_download_url -OutFile $archivePath -TimeoutSec 120
        Invoke-WebRequest -UseBasicParsing -Uri $checksumAsset[0].browser_download_url -OutFile $checksumPath -TimeoutSec 120

        $expected = @(foreach ($line in Get-Content -LiteralPath $checksumPath) {
            if ($line -match '^([0-9a-fA-F]{64})\s+\*?(?:\./)?(.+)$' -and $Matches[2] -ceq $archiveName) {
                $Matches[1]
            }
        })
        if ($expected.Count -ne 1) {
            throw "Missing, invalid, or duplicate checksum for $archiveName."
        }
        $actual = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash
        if ($actual -ne $expected[0]) {
            throw 'Downloaded archive failed SHA-256 verification.'
        }

        $extractDir = Join-Path $temporaryDir 'extracted'
        Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir
        $binaryPath = Join-Path $extractDir 'qrr.exe'
        if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
            throw 'Archive does not contain qrr.exe.'
        }
        $null = New-Item -ItemType Directory -Path $installDir -Force
        $stagedBinary = Join-Path $installDir ('.qrr-' + [guid]::NewGuid().ToString('N') + '.exe')
        Copy-Item -LiteralPath $binaryPath -Destination $stagedBinary
        Move-Item -LiteralPath $stagedBinary -Destination (Join-Path $installDir 'qrr.exe') -Force
        $stagedBinary = $null

        # Preserve existing entries and add the directory only to the user's PATH.
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        $userEntries = @($userPath -split ';' | Where-Object { $_ })
        $matchingEntries = @($userEntries | Where-Object {
            [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\', '/') -eq $installDir.TrimEnd('\', '/')
        })
        if ($matchingEntries.Count -eq 0) {
            $newUserPath = ($userEntries + $installDir) -join ';'
            [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
        }
        $processEntries = @($env:Path -split ';' | Where-Object { $_ })
        if ($processEntries -notcontains $installDir) {
            $env:Path = ($processEntries + $installDir) -join ';'
        }

        Write-Host "Installed qrr $($release.tag_name) to $(Join-Path $installDir 'qrr.exe')"
        Write-Host 'Run qrr --help to get started. Open a new terminal for the updated user PATH.'
    }
    finally {
        if ($stagedBinary -and (Test-Path -LiteralPath $stagedBinary)) {
            Remove-Item -LiteralPath $stagedBinary -Force
        }
        if ($temporaryDir -and (Test-Path -LiteralPath $temporaryDir)) {
            Remove-Item -LiteralPath $temporaryDir -Recurse -Force
        }
    }
}
