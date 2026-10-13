$ErrorActionPreference='Stop'
$os=Get-CimInstance Win32_OperatingSystem
$cpu=Get-CimInstance Win32_Processor | Select-Object -First 1
$admin=([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
Write-Host "OS: $($os.Caption) ($($os.Version))"
Write-Host "Architecture: $($os.OSArchitecture)"
Write-Host "Memory (GB): $([math]::Round($os.TotalVisibleMemorySize/1MB,1))"
Write-Host "VirtualizationFirmwareEnabled: $($cpu.VirtualizationFirmwareEnabled)"
Write-Host "Admin: $admin"
if (-not [Environment]::Is64BitOperatingSystem) { throw '64-bit Windows required.' }
if ($os.Version -notmatch '^10\.') { throw 'Windows 10/11 required.' }
if (-not $cpu.VirtualizationFirmwareEnabled) { Write-Warning 'Firmware virtualization flag is false; this is not conclusive when a hypervisor is active. Check Docker Engine and Task Manager CPU virtualization status.' }
if (-not $admin) { Write-Warning 'Administrator permissions are required for runtime installation.' }
try { wsl --status } catch { Write-Warning 'WSL is not yet available.' }
