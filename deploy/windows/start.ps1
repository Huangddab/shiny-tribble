$ErrorActionPreference='Stop'
Set-Location $PSScriptRoot
docker compose -f compose.yaml up -d --pull never
if ($LASTEXITCODE -ne 0) { throw 'Start failed' }
Start-Process 'http://127.0.0.1:8080/map'
