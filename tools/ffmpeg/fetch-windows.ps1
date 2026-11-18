# Fetch the Windows FFmpeg Essentials build from the vendor linked by FFmpeg.org.
# This script is used only for the Windows distribution build or when the user
# explicitly runs SETUP-FFmpeg.cmd. It does not run when AussieTV starts.
param(
  [string]$Destination = (Join-Path $PSScriptRoot '..\..\ffmpeg'),
  [string]$AppRoot = ''
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$uri = 'https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip'
$hashURI = "$uri.sha256"
$work = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $work -Force | Out-Null
try {
    $archive = Join-Path $work 'ffmpeg.zip'
    $hashFile = Join-Path $work 'ffmpeg.zip.sha256'
    Invoke-WebRequest -Uri $uri -OutFile $archive
    Invoke-WebRequest -Uri $hashURI -OutFile $hashFile
    $contents = Get-Content $hashFile -Raw
    $match = [regex]::Match($contents, '(?i)\b[a-f0-9]{64}\b')
    if (-not $match.Success) { throw 'Unable to parse the published FFmpeg SHA-256 checksum' }
    $expected = $match.Value.ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 -Path $archive).Hash.ToLowerInvariant()
    if ($actual -ne $expected) { throw "FFmpeg checksum mismatch: expected $expected, got $actual" }
    $expanded = Join-Path $work 'unpacked'
    Expand-Archive -Path $archive -DestinationPath $expanded
    $binary = @(Get-ChildItem -Path $expanded -Recurse -Filter 'ffmpeg.exe' -File)
    if ($binary.Count -ne 1) { throw "Expected exactly one ffmpeg.exe in the verified archive; found $($binary.Count)" }
    $binary = $binary[0]
    $dest = [IO.Path]::GetFullPath($Destination)
    New-Item -ItemType Directory -Path $dest -Force | Out-Null
    Copy-Item -LiteralPath $binary.FullName -Destination (Join-Path $dest 'ffmpeg.exe') -Force
    # Keep any upstream GPL/license files present in the archive as well as
    # the standalone license and provenance notice distributed with AussieTV.
    $licenseFiles = @(Get-ChildItem -Path $expanded -Recurse -File | Where-Object {
      $_.Name -match '^(LICENSE|COPYING)(\.[a-zA-Z0-9_-]+)?$'
    })
    foreach ($license in $licenseFiles) {
      Copy-Item -LiteralPath $license.FullName -Destination (Join-Path $dest ('upstream-' + $license.Name)) -Force
    }
    $versionLines = & (Join-Path $dest 'ffmpeg.exe') -version 2>&1
    $version = $versionLines | Select-Object -First 1
    if ($LASTEXITCODE -ne 0 -or -not $version) { throw 'Downloaded FFmpeg executable did not run on Windows' }
    @(
      "Upstream: $uri"
      "Archive SHA-256: $actual"
      "Verified version: $version"
      'Build provider: https://www.gyan.dev/ffmpeg/builds/'
      'Source and build scripts: https://github.com/FFmpeg/FFmpeg and https://github.com/GyanD/codexffmpeg'
      'License: GPLv3 for the static Gyan essentials build. See GPL-3.0.txt and upstream notices.'
    ) | Set-Content -Encoding UTF8 -Path (Join-Path $dest 'BUILD-INFO.txt')
    # Store the chosen location only after download, checksum and runtime checks pass.
    if ($AppRoot) {
      $absoluteRoot = [IO.Path]::GetFullPath($AppRoot)
      if (-not (Test-Path -LiteralPath $absoluteRoot -PathType Container)) {
        throw 'AussieTV installation directory does not exist'
      }
      $locationFile = Join-Path $absoluteRoot 'ffmpeg-location.txt'
      [IO.File]::WriteAllText($locationFile, (Join-Path $dest 'ffmpeg.exe') + [Environment]::NewLine, [Text.UTF8Encoding]::new($false))
    }
    Write-Host "Verified portable FFmpeg: $version"
    Write-Host "Installed to $dest"
} finally {
    Remove-Item -LiteralPath $work -Force -Recurse -ErrorAction SilentlyContinue
}
