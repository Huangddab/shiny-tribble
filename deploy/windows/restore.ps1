param([Parameter(Mandatory=$true)][string]$Archive)
$ErrorActionPreference='Stop'
Set-Location $PSScriptRoot
if (-not (Test-Path -LiteralPath $Archive)) { throw "Backup not found: $Archive" }
$resolved=(Resolve-Path -LiteralPath $Archive).Path
if (-not $resolved.EndsWith('.archive.gz')) { throw 'Expected a MongoDB .archive.gz backup file.' }
$confirm=Read-Host 'Restore MongoDB from backup? Existing records may be overwritten. Type RESTORE to continue'
if ($confirm -cne 'RESTORE') { throw 'Restore cancelled.' }
$mongo=(docker compose -f compose.yaml ps -q mongo)
if ($LASTEXITCODE -ne 0 -or -not $mongo) { throw 'MongoDB container is not running.' }
docker cp $resolved "${mongo}:/tmp/400c-restore.archive"
if ($LASTEXITCODE -ne 0) { throw 'Could not copy backup to MongoDB container.' }
try {
  docker exec $mongo mongorestore --archive=/tmp/400c-restore.archive --gzip --drop
  if ($LASTEXITCODE -ne 0) { throw 'MongoDB restore failed.' }
} finally {
  docker exec $mongo rm -f /tmp/400c-restore.archive | Out-Null
}
Write-Host 'MongoDB restore completed. Restart the platform to refresh application state.'
