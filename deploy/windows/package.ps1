param([string]$OutputDirectory = '')
$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
if (-not $OutputDirectory) { $OutputDirectory=Join-Path $root 'dist/400C-Platform-v1.0' }
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
Copy-Item (Join-Path $PSScriptRoot '*.ps1') $OutputDirectory -Force
Copy-Item (Join-Path $PSScriptRoot '*.cmd') $OutputDirectory -Force
Copy-Item (Join-Path $PSScriptRoot 'INSTALLATION.md') $OutputDirectory -Force
New-Item -ItemType Directory -Force -Path (Join-Path $OutputDirectory 'runtime') | Out-Null
foreach ($runtimeFile in @('DockerDesktopInstaller.exe','wsl-installer.msi')) {
  $source=Join-Path $PSScriptRoot "runtime/$runtimeFile"
  if (Test-Path $source) { Copy-Item $source (Join-Path $OutputDirectory "runtime/$runtimeFile") -Force }
}
Copy-Item (Join-Path $PSScriptRoot 'compose.yaml') $OutputDirectory -Force
Copy-Item (Join-Path $PSScriptRoot 'README.md') $OutputDirectory -Force
New-Item -ItemType Directory -Force -Path (Join-Path $OutputDirectory 'images'),(Join-Path $OutputDirectory 'data/tileserver'),(Join-Path $OutputDirectory 'data/mosquitto') | Out-Null
$map=Join-Path $root 'docker/data/tileserver/fuzhou.mbtiles'
$conf=Join-Path $root 'docker/data/mosquitto/mosquitto.conf'
if (-not (Test-Path $map)) { throw "Missing MBTiles: $map" }
if (-not (Test-Path $conf)) { throw "Missing Mosquitto config: $conf" }
Copy-Item $map (Join-Path $OutputDirectory 'data/tileserver/fuzhou.mbtiles')
Copy-Item $conf (Join-Path $OutputDirectory 'data/mosquitto/mosquitto.conf')
$archive=Join-Path $OutputDirectory 'images/400c-images.tar'
docker save -o $archive chem-platform:latest mongo:8.0 eclipse-mosquitto:2 it.enboso.com:8080/wm/tileserver-gl:latest
if ($LASTEXITCODE -ne 0) { throw 'docker save failed' }
$hashes=@('images/400c-images.tar','data/tileserver/fuzhou.mbtiles','data/mosquitto/mosquitto.conf') | ForEach-Object {
  $hash=(Get-FileHash -Algorithm SHA256 (Join-Path $OutputDirectory $_)).Hash.ToLowerInvariant()
  "$hash  $_"
}
$hashes | Set-Content -Encoding ascii (Join-Path $OutputDirectory 'SHA256SUMS.txt')
Write-Host "Offline package ready: $OutputDirectory"
