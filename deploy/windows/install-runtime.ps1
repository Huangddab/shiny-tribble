param([switch]$SkipWSL)
$ErrorActionPreference='Stop'
Set-Location $PSScriptRoot
$admin=([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $admin) { throw 'Run this script as Administrator.' }
& (Join-Path $PSScriptRoot 'check-environment.ps1')
if (-not $SkipWSL) {
  Write-Host 'Enabling Windows WSL and VirtualMachinePlatform optional features...'
  foreach ($feature in @('Microsoft-Windows-Subsystem-Linux','VirtualMachinePlatform')) {
    $state=(Get-WindowsOptionalFeature -Online -FeatureName $feature).State
    if ($state -ne 'Enabled') {
      Enable-WindowsOptionalFeature -Online -FeatureName $feature -All -NoRestart | Out-Null
      Write-Warning 'Windows reboot may be required before proceeding.'
    }
  }
  $wslMsi=Join-Path $PSScriptRoot 'runtime/wsl-installer.msi'
  if (Test-Path $wslMsi) {
    $p=Start-Process msiexec.exe -ArgumentList @('/i',"`"$wslMsi`"","/passive","/norestart") -Wait -PassThru
    if ($p.ExitCode -notin @(0,3010)) { throw "WSL MSI installation failed: $($p.ExitCode)" }
    if ($p.ExitCode -eq 3010) { Write-Warning 'Reboot required after WSL installation.' }
  } else {
    Write-Warning 'No offline WSL MSI found. Provide runtime/wsl-installer.msi if WSL is not installed.'
  }
}
$dockerInstaller=Join-Path $PSScriptRoot 'runtime/DockerDesktopInstaller.exe'
if (-not (Test-Path $dockerInstaller)) { throw "Missing Docker Desktop installer: $dockerInstaller" }
Write-Host 'Launching official Docker Desktop installer. Follow its license prompts.'
$p=Start-Process -FilePath $dockerInstaller -ArgumentList @('install','--backend=wsl-2') -Wait -PassThru
if ($p.ExitCode -ne 0) { throw "Docker Desktop installer returned $($p.ExitCode)" }
Write-Host 'Restart Windows if prompted. Launch Docker Desktop, accept license terms, then run install.ps1.'
