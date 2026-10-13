$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
Write-Host '400C offline installer'
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker Desktop is not installed. Install and start licensed Docker Desktop first.' }
docker info *> $null
if ($LASTEXITCODE -ne 0) { throw 'Docker Engine is not running. Start Docker Desktop and select Linux containers.' }
docker compose version
if ($LASTEXITCODE -ne 0) { throw 'Docker Compose v2 is required.' }
if (-not (Test-Path './images/400c-images.tar')) { throw 'Missing images/400c-images.tar' }
if (-not (Test-Path './data/tileserver/fuzhou.mbtiles')) { throw 'Missing data/tileserver/fuzhou.mbtiles' }
if (-not (Test-Path './data/mosquitto/mosquitto.conf')) { throw 'Missing data/mosquitto/mosquitto.conf' }
if (Test-Path './SHA256SUMS.txt') {
  Get-Content './SHA256SUMS.txt' | ForEach-Object {
    if ($_ -match '^([0-9a-fA-F]{64})\s+(.+)$') {
      $expected=$Matches[1]; $file=$Matches[2]
      if (-not (Test-Path $file)) { throw "Missing checksum file: $file" }
      if ((Get-FileHash -Algorithm SHA256 $file).Hash -ne $expected) { throw "Checksum mismatch: $file" }
    }
  }
}
docker load -i './images/400c-images.tar'
if ($LASTEXITCODE -ne 0) { throw 'Docker image import failed.' }
docker compose -f compose.yaml up -d --pull never
if ($LASTEXITCODE -ne 0) { throw 'Service startup failed.' }
Write-Host 'Waiting for HTTP service...'
$ready=$false
for ($i=0; $i -lt 60; $i++) {
  try {
    $r=Invoke-WebRequest -Uri 'http://127.0.0.1:8080/map' -UseBasicParsing -TimeoutSec 3
    if ($r.StatusCode -eq 200) { $ready=$true; break }
  } catch {}
  Start-Sleep -Seconds 2
}
docker compose -f compose.yaml ps
if (-not $ready) { throw 'Web readiness check failed; inspect docker compose logs.' }
Write-Host '400C is ready: http://127.0.0.1:8080/map'
