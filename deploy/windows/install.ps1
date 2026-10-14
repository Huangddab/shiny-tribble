$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
Write-Host '400C offline installer'
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { throw 'Docker Desktop is not installed. Install and start licensed Docker Desktop first.' }
& cmd.exe /d /c 'docker info >nul 2>&1'
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
Write-Host 'Checking initial administrator...'
$statusOutput = @(docker compose -f compose.yaml run --rm --no-deps chem-platform user status 2>&1 | Out-String)
if ($LASTEXITCODE -ne 0) { throw "Administrator status check failed: $statusOutput" }
if ($statusOutput -match 'INITIAL_SETUP_REQUIRED') {
  Write-Host 'No administrator exists. Set the initial admin password (minimum 12 characters, maximum 72 UTF-8 bytes).'
  $secret = Read-Host 'New admin password' -AsSecureString
  $confirm = Read-Host 'Confirm admin password' -AsSecureString
  $ptr1 = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secret)
  $ptr2 = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($confirm)
  try {
    $pass1 = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr1)
    $pass2 = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr2)
    if ($pass1 -cne $pass2) { throw 'Passwords do not match.' }
    $byteCount = [Text.Encoding]::UTF8.GetByteCount($pass1)
    if ($pass1.Length -lt 12 -or $byteCount -gt 72) { throw 'Password must be at least 12 characters and no more than 72 UTF-8 bytes.' }
    $env:ADMIN_PASSWORD = $pass1
    try {
      docker compose -f compose.yaml run --rm --no-deps -e ADMIN_PASSWORD chem-platform user init
      if ($LASTEXITCODE -ne 0) { throw 'Administrator initialization failed.' }
    } finally {
      Remove-Item Env:ADMIN_PASSWORD -ErrorAction SilentlyContinue
    }
  } finally {
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr1)
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr2)
    Remove-Variable pass1,pass2 -ErrorAction SilentlyContinue
  }
} elseif ($statusOutput -match 'ADMIN_EXISTS') {
  Write-Host 'Existing administrator found; password and user data preserved.'
} else {
  throw "Unrecognized administrator status: $statusOutput"
}
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
