$ErrorActionPreference='Continue'
Set-Location $PSScriptRoot
$stamp=Get-Date -Format 'yyyyMMdd-HHmmss'
$dir=Join-Path $PSScriptRoot "diagnostics/$stamp"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
docker version *> (Join-Path $dir 'docker-version.txt')
docker compose -f compose.yaml ps -a *> (Join-Path $dir 'services.txt')
docker compose -f compose.yaml logs --no-color --tail 300 *> (Join-Path $dir 'services.log')
try {
  $r=Invoke-WebRequest 'http://127.0.0.1:8080/map' -UseBasicParsing -TimeoutSec 5
  "HTTP /map: $($r.StatusCode)" | Set-Content (Join-Path $dir 'http.txt')
} catch { "HTTP /map failed: $_" | Set-Content (Join-Path $dir 'http.txt') }
Write-Warning 'Diagnostic logs may contain device identifiers, locations or secrets. Review before sharing.'
Write-Host "Diagnostics saved: $dir"
