# Run in an elevated PowerShell. Forwards Windows LAN ports to WSL Vite / Vue CLI.
$ErrorActionPreference = "Stop"

$wslIp = (wsl -e hostname -I).Trim().Split(" ", [System.StringSplitOptions]::RemoveEmptyEntries) |
    Where-Object { $_ -like "172.*" -or $_ -like "192.*" } |
    Select-Object -First 1
if (-not $wslIp) {
    throw "Could not detect WSL IPv4. Is WSL running?"
}

$lanIp = Get-NetIPAddress -AddressFamily IPv4 |
    Where-Object {
        $_.IPAddress -notlike "127.*" -and
        $_.IPAddress -notlike "172.2*" -and
        $_.InterfaceAlias -match "WLAN|Wi-?Fi|无线|以太网|Ethernet"
    } |
    Select-Object -ExpandProperty IPAddress -First 1

$ports = @(
    @{ Port = 5185; Name = "CIGC dapp 5185" },
    @{ Port = 8081; Name = "CIGC admin 8081" }
)
foreach ($p in $ports) {
    netsh interface portproxy delete v4tov4 listenport=$($p.Port) listenaddress=0.0.0.0 | Out-Null
    netsh interface portproxy add v4tov4 listenport=$($p.Port) listenaddress=0.0.0.0 connectport=$($p.Port) connectaddress=$wslIp
    netsh advfirewall firewall delete rule name="$($p.Name)" | Out-Null
    netsh advfirewall firewall add rule name="$($p.Name)" dir=in action=allow protocol=TCP localport=$($p.Port) | Out-Null
}

Write-Host "WSL:   $wslIp"
Write-Host "Dapp:  http://${lanIp}:5185/"
Write-Host "Admin: http://${lanIp}:8081/admin"
netsh interface portproxy show all
