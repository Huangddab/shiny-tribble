# 400C Windows 10/11 offline delivery (V1.0)

## Prerequisites
- x64 Windows 10/11 supported by the installed Docker Desktop release, virtualization enabled, WSL2 available.
- Licensed Docker Desktop (government customer deployments require applicable paid subscription).
- Docker Desktop started in Linux containers mode before installation.
- Sufficient free disk space for the image archive, extracted images, MBTiles and database.
- Ports 8080 and 1883 available; allow only trusted LAN access.

## Build delivery package on development PC
1. Build the latest application image: `docker build -t chem-platform:latest .`
2. Ensure MongoDB 8.0, Eclipse Mosquitto 2 and the private TileServer GL image are locally present.
3. Place `fuzhou.mbtiles` under `docker/data/tileserver/` and `mosquitto.conf` under `docker/data/mosquitto/`.
4. Run `powershell -ExecutionPolicy Bypass -File deploy/windows/package.ps1` from the repository root.
5. Copy the complete `dist/400C-Platform-v1.0` folder to the offline customer PC.

## Install / operate
- Install WSL2 and a licensed Docker Desktop from trusted offline installers, accept license terms, then launch Docker Desktop.
- In the delivery directory, run `powershell -ExecutionPolicy Bypass -File .\install.ps1`.
- To start: `powershell -ExecutionPolicy Bypass -File .\start.ps1`.
- To stop: `powershell -ExecutionPolicy Bypass -File .\stop.ps1`.
- To back up MongoDB: `powershell -ExecutionPolicy Bypass -File .\backup.ps1`.
- Browser: `http://127.0.0.1:8080/map`, or trusted LAN IP on port 8080.

## Known V1.0 limitations
- Docker Desktop must be running; unattended pre-login boot is NOT guaranteed.
- `backup.ps1` covers MongoDB only, not MQTT volumes or configuration.
- Images currently use upstream version tags and `chem-platform:latest`; release reproducibility requires digest pinning.
- Review MQTT authentication and permissions before deployment. Do not expose MQTT to untrusted networks.
- Validate third-party software, maps and Docker Desktop redistribution/licensing independently.
- `install.ps1` checks HTTP /map; validate real MQTT device telemetry and tile loading during acceptance.
