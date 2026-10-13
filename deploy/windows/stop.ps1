$ErrorActionPreference='Stop'
Set-Location $PSScriptRoot
docker compose -f compose.yaml stop
if ($LASTEXITCODE -ne 0) { throw 'Stop failed' }
