param(
  [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

$Repo = "Encratahq/encrata-cli"
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\Encrata"
$ExePath = Join-Path $InstallDir "encrata.exe"

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

if ($Version -eq "latest") {
  $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
  $tag = $release.tag_name
  $versionNumber = $tag.TrimStart("v")
} else {
  $versionNumber = $Version.TrimStart("v")
  $tag = "v$versionNumber"
}

$asset = "encrata_${versionNumber}_windows_${arch}.zip"
$url = "https://github.com/$Repo/releases/download/$tag/$asset"

$tempDir = Join-Path $env:TEMP "encrata-install"
$zipPath = Join-Path $tempDir $asset

New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Write-Host "Downloading Encrata CLI $tag for windows/$arch..."
Invoke-WebRequest -Uri $url -OutFile $zipPath

Write-Host "Installing to $InstallDir..."
Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$InstallDir*") {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
  $env:Path = "$env:Path;$InstallDir"
  Write-Host "Added Encrata to your user PATH. Open a new PowerShell window to use it."
}

Write-Host "Installed:"
& $ExePath version
