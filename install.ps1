$ErrorActionPreference = 'Stop'

$repo = "Lucu-lucuan-Lab/nolife-cli"
$appName = "nolife"
$installDir = Join-Path $env:LOCALAPPDATA "Programs\$appName"
$exePath = Join-Path $installDir "$appName.exe"
$githubUrl = "https://github.com/$repo"
$apiUrl = "https://api.github.com/repos/$repo/releases/latest"

function Write-Step($msg) {
    Write-Host "`n[*] $msg" -ForegroundColor Cyan
}

function Write-Success($msg) {
    Write-Host "[+] $msg" -ForegroundColor Green
}

function Write-WarningMsg($msg) {
    Write-Host "[!] $msg" -ForegroundColor Yellow
}

function Write-ErrorMsg($msg) {
    Write-Host "[X] $msg" -ForegroundColor Red
}

function Get-Architecture {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { return "arm64" }
    return "amd64"
}

function Check-Chrome {
    $paths = @(
        "${env:ProgramFiles}\Google\Chrome\Application\chrome.exe",
        "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe",
        "${env:LocalAppData}\Google\Chrome\Application\chrome.exe"
    )
    
    foreach ($path in $paths) {
        if (Test-Path $path) { return $true }
    }
    
    # Check Registry as fallback
    $regPaths = @(
        "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe",
        "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe",
        "HKCU:\Software\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe"
    )
    
    foreach ($reg in $regPaths) {
        if (Test-Path $reg) { return $true }
    }
    
    return $false
}

function Assert-NotRunning {
    $proc = Get-Process -Name $appName -ErrorAction SilentlyContinue
    if ($proc) {
        Write-ErrorMsg "$appName is currently running. Please close it before installing/updating."
        exit 1
    }
}

function Add-ToPath($pathToAdd) {
    $registryPath = "HKCU:\Environment"
    $name = "Path"
    
    $currentPathRaw = (Get-ItemProperty -Path $registryPath -Name $name -ErrorAction SilentlyContinue).Path
    if ($null -eq $currentPathRaw) { $currentPathRaw = "" }
    
    $normalizedNew = $pathToAdd.TrimEnd('\', '/').Replace('/', '\')
    $pathEntries = $currentPathRaw -split ";" | Where-Object { $_ -ne "" }
    
    $alreadyExists = $false
    foreach ($entry in $pathEntries) {
        if ($entry.TrimEnd('\', '/').Replace('/', '\') -eq $normalizedNew) {
            $alreadyExists = $true
            break
        }
    }
    
    if (-not $alreadyExists) {
        $newPath = if ($currentPathRaw -and -not $currentPathRaw.EndsWith(";")) { "$currentPathRaw;$pathToAdd" } else { "$currentPathRaw$pathToAdd" }
        Set-ItemProperty -Path $registryPath -Name $name -Value $newPath -Type ExpandString
        Write-Success "Added $pathToAdd to User PATH (Registry preserved)."
    }
}

try {
    Write-Step "Checking for $appName process..."
    Assert-NotRunning

    Write-Step "Detecting architecture..."
    $arch = Get-Architecture
    Write-Host "Detected: $arch"

    Write-Step "Fetching latest release from GitHub..."
    $release = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
    $version = $release.tag_name
    Write-Host "Version: $version"

    $assetName = "$appName-windows-$arch.exe"
    $asset = $release.assets | Where-Object { $_.name -eq $assetName }
    if (-not $asset) { throw "Could not find asset $assetName in release $version" }

    $checksumName = "$assetName.sha256"
    $checksumAsset = $release.assets | Where-Object { $_.name -eq $checksumName }

    # Create install directory
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    $tempExe = Join-Path $env:TEMP "$assetName"
    
    Write-Step "Downloading $appName..."
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $tempExe -UseBasicParsing

    if ($checksumAsset) {
        Write-Step "Verifying integrity..."
        $expectedHash = (Invoke-RestMethod -Uri $checksumAsset.browser_download_url -UseBasicParsing).Split(" ")[0].Trim().ToLower()
        $actualHash = (Get-FileHash $tempExe -Algorithm SHA256).Hash.ToLower()
        
        if ($expectedHash -ne $actualHash) {
            Remove-Item $tempExe -ErrorAction SilentlyContinue
            throw "Integrity check failed! Expected: $expectedHash, Actual: $actualHash"
        }
        Write-Success "Checksum verified."
    } else {
        Write-WarningMsg "No checksum found for this release. Skipping verification."
    }

    Write-Step "Installing..."
    Move-Item -Path $tempExe -Destination $exePath -Force
    
    Add-ToPath $installDir

    Write-Step "Checking dependencies..."
    if (-not (Check-Chrome)) {
        Write-WarningMsg "Google Chrome was not detected on your system."
        Write-WarningMsg "Nolife needs Chrome/Chromium to navigate anime sites."
        Write-WarningMsg "Please download it from: https://www.google.com/chrome/"
    } else {
        Write-Success "Google Chrome detected."
    }

    Write-Success "Successfully installed $appName $version!"
    Write-Host "`nUsage:" -ForegroundColor Gray
    Write-Host "  $appName download [URL]"
    Write-Host "  $appName search [query]"
    Write-Host "`nNote: You may need to restart your terminal to use the '$appName' command." -ForegroundColor Gray

} catch {
    Write-ErrorMsg "Installation failed: $($_.Exception.Message)"
    exit 1
}
