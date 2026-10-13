$ErrorActionPreference='Stop'
Set-Location $PSScriptRoot
Write-Host '400C Windows offline setup'
& (Join-Path $PSScriptRoot 'check-environment.ps1')
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  Write-Host 'Docker Desktop not found. Runtime installation is required.'
  $admin=([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
  if (-not $admin) { throw 'Restart setup.ps1 in an elevated PowerShell session.' }
  & (Join-Path $PSScriptRoot 'install-runtime.ps1')
  Write-Warning 'Complete any required Windows reboot, open Docker Desktop and accept license terms. Then rerun setup.ps1.'
  exit 0
}
docker info *> $null
if ($LASTEXITCODE -ne 0) {
  Write-Warning 'Docker Desktop is installed but its Linux engine is not running. Open Docker Desktop and complete first-run initialization, then rerun setup.ps1.'
  exit 1
}
& (Join-Path $PSScriptRoot 'install.ps1')
if (-not $?) { throw 'Platform installation failed.' }
