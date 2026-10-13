# Windows 10/11 customer installation (offline)

## Licensing and prerequisites
Docker Desktop is separately licensed. Government use requires an applicable paid subscription. Obtain the installer from an authorized source and confirm redistribution rights; this repository does not contain the installer.
Ensure BIOS virtualization is enabled, the OS version is supported by the chosen Docker Desktop version, WSL2 prerequisites are met, and the machine has enough memory and disk capacity.

## Prepare the package
Build the Go image and run `package.ps1` on the development computer.
Separately obtain a compatible official Docker Desktop installer and WSL MSI from official sources. Place them in the package under `runtime/DockerDesktopInstaller.exe` and `runtime/wsl-installer.msi`.
For fully offline WSL provisioning, verify all required Windows components and dependencies are available in the customer Windows image. The WSL MSI alone may not satisfy every OS configuration.

## Customer setup
1. Right-click PowerShell, select Run as administrator.
2. Run `powershell -ExecutionPolicy Bypass -File .\check-environment.ps1`.
3. Run `powershell -ExecutionPolicy Bypass -File .\install-runtime.ps1`.
4. Restart Windows if required.
5. Open Docker Desktop and complete first-run license acceptance and initialization. Confirm it is using Linux containers.
6. Run `powershell -ExecutionPolicy Bypass -File .\install.ps1`.
7. Open `http://127.0.0.1:8080/map`. Verify map tiles and a real MQTT device connection.

## Limitations
This is a scripted installer, not yet a single EXE or a fully automated reboot-resume installer.
WSL2/Docker Desktop startup may require a user session. Government licensing and third-party redistributable rights must be resolved by the customer/deployer.
