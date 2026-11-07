# 用法: .\install.ps1 [-Version 0.1.0] [-InstallDir C:\Tools\bin]
param(
    [string]$Version = "",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\fan"
)

$ErrorActionPreference = "Stop"

$Owner = "china-lang-fan"
$Repo = "fan-go"
$Binary = "fan.exe"

function Get-Architecture {
    switch -Wildcard ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { "amd64" }
        "ARM64" { "arm64" }
        default { throw "不支持的架构：$env:PROCESSOR_ARCHITECTURE" }
    }
}

$Arch = Get-Architecture

if (-not $Version) {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Owner/$Repo/releases/latest" -Headers @{ "User-Agent" = "fan-install" }
    $Version = $release.tag_name.TrimStart("v")
}

$Name = "fan_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Owner/$Repo/releases/download/v${Version}/$Name"

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $TempDir | Out-Null

try {
    $ZipPath = Join-Path $TempDir $Name
    Write-Host "下载 $Url"
    Invoke-WebRequest -Uri $Url -OutFile $ZipPath
    Expand-Archive -Path $ZipPath -DestinationPath $TempDir

    $BinaryPath = Join-Path $TempDir $Binary
    if (-not (Test-Path $BinaryPath)) {
        throw "压缩包中未找到 $Binary"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path $BinaryPath -Destination (Join-Path $InstallDir $Binary) -Force

    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
        Write-Host "已将 $InstallDir 加入用户 PATH，重新打开终端后生效"
    }

    Write-Host "安装完成：$(Join-Path $InstallDir $Binary)"
    & (Join-Path $InstallDir $Binary) version
}
finally {
    Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}
