param(
    [string]$Version
)

$ErrorActionPreference = "Stop"

$Repo = "Rikkiuuu/hop"

if ([string]::IsNullOrWhiteSpace($Version)) {
    try {
        $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "hop-installer" }
        $Version = $Release.tag_name
    } catch {
        $Version = "v1.0.0"
    }
}

if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = "v1.0.0"
}

$Ver = $Version.TrimStart("v")
$Version = "v$Ver"

$RawArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToUpperInvariant()
if ([string]::IsNullOrWhiteSpace($RawArch)) {
    $RawArch = $env:PROCESSOR_ARCHITECTURE.ToUpperInvariant()
}

switch ($RawArch) {
    "AMD64" { $Arch = "amd64" }
    "X64"   { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default { throw "Unsupported architecture: $RawArch" }
}

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Force -Path $TempDir | Out-Null

try {
    $Archive = "hop_${Ver}_windows_${Arch}.zip"
    $Url = "https://github.com/$Repo/releases/download/$Version/$Archive"
    $ZipPath = Join-Path $TempDir $Archive

    Invoke-WebRequest -Uri $Url -OutFile $ZipPath
    Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force

    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\hop"
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Path (Join-Path $TempDir "hop.exe") -Destination (Join-Path $InstallDir "hop.exe") -Force

    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $PathEntries = if ($UserPath) { $UserPath -split ";" | Where-Object { $_ -ne "" } } else { @() }
    if ($PathEntries -notcontains $InstallDir) {
        $NewUserPath = ($PathEntries + $InstallDir) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $NewUserPath, "User")
        $env:Path = "$env:Path;$InstallDir"
    }

    Write-Output "Installed hop $Version to $InstallDir\hop.exe"
} finally {
    if (Test-Path $TempDir) {
        Remove-Item -Recurse -Force $TempDir
    }
}
