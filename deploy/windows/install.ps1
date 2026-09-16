#Requires -RunAsAdministrator
<#
.SYNOPSIS
  Installiert den CloudWeb-Druckertreiber + Dienst unter Windows.
.DESCRIPTION
  Siehe doc/anforderung_windows.md. Legt an bzw. aktualisiert idempotent:
    - C:\ProgramData\printtoqrview - Konfiguration (backend.env), Binaries
      und das feste Druckziel (incoming.pdf).
    - Drucker "CloudWeb" mit dem vorhandenen Inbox-Treiber
      "Microsoft Print To PDF", gebunden an einen festen Datei-Port statt
      PORTPROMPT: - der originale Drucker "Microsoft Print to PDF" bleibt
      dabei unverändert (live verifiziert, siehe doc/anforderung_windows.md).
    - Windows-Dienst "CloudWeb" (cloudweb.exe: HTTP-Anzeige + Ordner-Watcher
      auf incoming.pdf), per golang.org/x/sys/windows/svc - kein NSSM nötig.
    - Explorer-"Senden an"-Eintrag "CloudWeb Share" (fileshare.exe).
  Erwartet cloudweb.exe und fileshare.exe im selben Verzeichnis wie dieses
  Skript (z. B. nach dem Kopieren von einer Build-Freigabe).
#>

$ErrorActionPreference = "Stop"

$DataDir = "C:\ProgramData\printtoqrview"
$PortPath = Join-Path $DataDir "incoming.pdf"
$EnvPath = Join-Path $DataDir "backend.env"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$CloudwebExe = Join-Path $DataDir "cloudweb.exe"
$FileshareExe = Join-Path $DataDir "fileshare.exe"

foreach ($exe in "cloudweb.exe", "fileshare.exe") {
    if (-not (Test-Path (Join-Path $ScriptDir $exe))) {
        throw "$exe fehlt neben install.ps1 ($ScriptDir) - erst per 'GOOS=windows GOARCH=amd64 go build' erzeugen."
    }
}

New-Item -ItemType Directory -Force -Path $DataDir | Out-Null

Write-Host "Kopiere Binaries nach $DataDir ..."
Copy-Item -Path (Join-Path $ScriptDir "cloudweb.exe") -Destination $CloudwebExe -Force
Copy-Item -Path (Join-Path $ScriptDir "fileshare.exe") -Destination $FileshareExe -Force

if (-not (Test-Path $EnvPath)) {
    Copy-Item -Path (Join-Path $ScriptDir "backend.env.example") -Destination $EnvPath
    Write-Warning "Neue Konfigurationsdatei angelegt: $EnvPath - bitte NC_BASE_URL/NC_USERNAME/NC_PASSWORD eintragen, danach: Restart-Service CloudWeb"
}

Write-Host "Richte Drucker 'CloudWeb' ein ..."
if (-not (Get-PrinterPort -Name $PortPath -ErrorAction SilentlyContinue)) {
    Add-PrinterPort -Name $PortPath
}
if (-not (Get-Printer -Name "CloudWeb" -ErrorAction SilentlyContinue)) {
    Add-Printer -Name "CloudWeb" -DriverName "Microsoft Print To PDF" -PortName $PortPath
}

Write-Host "Richte Dienst 'CloudWeb' ein ..."
$existing = Get-Service -Name "CloudWeb" -ErrorAction SilentlyContinue
if ($existing) {
    Stop-Service -Name "CloudWeb" -Force -ErrorAction SilentlyContinue
    sc.exe delete CloudWeb | Out-Null
    Start-Sleep -Seconds 1
}
New-Service -Name "CloudWeb" -BinaryPathName $CloudwebExe -DisplayName "CloudWeb (printtoqrview)" -StartupType Automatic | Out-Null
Start-Service -Name "CloudWeb"

Write-Host "Richte Explorer-'Senden an'-Eintrag ein ..."
$sendTo = Join-Path $env:APPDATA "Microsoft\Windows\SendTo"
$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut((Join-Path $sendTo "CloudWeb Share.lnk"))
$shortcut.TargetPath = $FileshareExe
$shortcut.Save()

Write-Host ""
Write-Host "Fertig. Zur Kontrolle:"
Write-Host "  Get-Service CloudWeb"
Write-Host "  Get-Printer | Select Name, DriverName, PortName"
Write-Host "  http://localhost:40080/"
