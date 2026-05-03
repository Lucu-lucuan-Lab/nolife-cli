$ErrorActionPreference = 'Stop'

$appName = "nolife"
$installDir = Join-Path $env:LOCALAPPDATA "Programs\$appName"

function Write-Step($msg) {
    Write-Host "`n[*] $msg" -ForegroundColor Cyan
}

function Write-Success($msg) {
    Write-Host "[+] $msg" -ForegroundColor Green
}

function Write-ErrorMsg($msg) {
    Write-Host "[X] $msg" -ForegroundColor Red
}

function Assert-NotRunning {
    $proc = Get-Process -Name $appName -ErrorAction SilentlyContinue
    if ($proc) {
        Write-ErrorMsg "$appName is currently running. Please close it before uninstalling."
        exit 1
    }
}

function Remove-FromPath($pathToRemove) {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
    $pathEntries = $currentPath -split ";" | Where-Object { $_ -ne "" }
    
    $normalizedPath = $pathToRemove.TrimEnd('\')
    $newEntries = @()
    $found = $false
    
    foreach ($entry in $pathEntries) {
        if ($entry.TrimEnd('\') -eq $normalizedPath) {
            $found = $true
        } else {
            $newEntries += $entry
        }
    }
    
    if ($found) {
        $newPathString = $newEntries -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newPathString, [EnvironmentVariableTarget]::User)
        Write-Success "Removed $pathToRemove from User PATH."
    }
}

try {
    Write-Step "Checking for $appName process..."
    Assert-NotRunning

    Write-Step "Removing from PATH..."
    Remove-FromPath $installDir

    Write-Step "Deleting files..."
    if (Test-Path $installDir) {
        Remove-Item -Path $installDir -Recurse -Force
        Write-Success "Deleted $installDir"
    } else {
        Write-Host "Installation directory not found. Already uninstalled?" -ForegroundColor Gray
    }

    Write-Success "Successfully uninstalled $appName."
    Write-Host "Note: You may need to restart your terminal for changes to take effect." -ForegroundColor Gray

} catch {
    Write-ErrorMsg "Uninstallation failed: $($_.Exception.Message)"
    exit 1
}
