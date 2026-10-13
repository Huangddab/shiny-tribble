$ErrorActionPreference='Stop'
Set-Location $PSScriptRoot
$stamp=Get-Date -Format 'yyyyMMdd-HHmmss'
$dir=Join-Path $PSScriptRoot "backups/$stamp"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$mongo=(docker compose -f compose.yaml ps -q mongo)
if ($LASTEXITCODE -ne 0 -or -not $mongo) { throw 'MongoDB container is not running' }
docker exec $mongo sh -c 'mongodump --archive=/tmp/400c-mongo.archive --gzip'
if ($LASTEXITCODE -ne 0) { throw 'mongodump failed' }
docker cp "${mongo}:/tmp/400c-mongo.archive" (Join-Path $dir 'mongo.archive.gz')
if ($LASTEXITCODE -ne 0) { throw 'Backup copy failed' }
docker exec $mongo rm -f /tmp/400c-mongo.archive | Out-Null
Write-Host "MongoDB backup created: $dir"
Write-Warning 'This script backs up MongoDB only. MQTT persistent state and configuration require separate backup.'
