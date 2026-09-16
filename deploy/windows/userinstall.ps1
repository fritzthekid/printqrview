<#
.SYNOPSIS
  Richtet den Explorer-"Senden an"-Eintrag "CloudWeb Share" ein.
.DESCRIPTION
  Bewusst getrennt von install.ps1 (das Admin-Rechte braucht): eine
  elevierte PowerShell kann unter einem anderen Benutzerprofil laufen
  (%APPDATA%) als der Nutzer, der später tatsächlich "Senden an" benutzt -
  live so vorgefunden (siehe GitHub-Issue #7). Deshalb ohne Admin-Rechte,
  in einer ganz normalen PowerShell des eigenen Logins ausführen:

    .\userinstall.ps1

  Setzt voraus, dass install.ps1 vorher (einmalig, als Administrator)
  gelaufen ist - C:\ProgramData\printtoqrview\fileshare.exe muss existieren.
#>

$ErrorActionPreference = "Stop"

$FileshareExe = "C:\ProgramData\printtoqrview\fileshare.exe"
if (-not (Test-Path $FileshareExe)) {
    throw "$FileshareExe fehlt - zuerst install.ps1 als Administrator ausfuehren."
}

$sendTo = Join-Path $env:USERPROFILE "AppData\Roaming\Microsoft\Windows\SendTo"
$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut((Join-Path $sendTo "CloudWeb Share.lnk"))
$shortcut.TargetPath = $FileshareExe
$shortcut.Save()

Write-Host "Fertig: 'Senden an' -> 'CloudWeb Share' sollte jetzt im Explorer-Kontextmenu erscheinen."
