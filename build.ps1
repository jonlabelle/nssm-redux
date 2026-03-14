[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("help", "build", "build-windows", "build-windows-amd64", "build-windows-arm64", "test", "vet", "lint", "fmt", "clean")]
    [string[]]$Task = @("help"),

    [string]$App = "nssmr",
    [string]$Source = "./cmd/nssmr",
    [string]$Packages = "./...",
    [string]$Bin = (Join-Path $PSScriptRoot "bin"),
    [string]$Dist = (Join-Path $PSScriptRoot "dist"),
    [string]$GoCache = (Join-Path $PSScriptRoot ".gocache"),
    [string]$GoModCache = (Join-Path $PSScriptRoot ".gomodcache"),

    [ValidateSet("amd64", "arm64")]
    [string[]]$WindowsArch = @("amd64", "arm64"),

    [string]$Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-ProcessEnvironmentValue {
    param([Parameter(Mandatory)][string]$Name)

    return [System.Environment]::GetEnvironmentVariable($Name, "Process")
}

function Set-ProcessEnvironmentValue {
    param(
        [Parameter(Mandatory)][string]$Name,
        [AllowNull()][string]$Value
    )

    if ([string]::IsNullOrEmpty($Value)) {
        Remove-Item -Path "Env:$Name" -ErrorAction SilentlyContinue
        return
    }

    Set-Item -Path "Env:$Name" -Value $Value
}

function Get-BuildVersion {
    param([string]$RequestedVersion)

    if ($RequestedVersion) {
        return $RequestedVersion
    }

    $git = Get-Command git -ErrorAction SilentlyContinue
    if ($git) {
        try {
            $description = & $git.Source describe --tags --always --dirty 2>$null
            if ($LASTEXITCODE -eq 0 -and $description) {
                return $description.Trim()
            }
        }
        catch {
        }
    }

    return "dev"
}

$script:BuildVersion = Get-BuildVersion -RequestedVersion $Version
$script:LinkerFlags = "-s -w -X github.com/jonlabelle/nssm-redux/internal/cli.Version=$($script:BuildVersion)"

function Invoke-Go {
    param(
        [Parameter(Mandatory)][string[]]$Arguments,
        [hashtable]$Environment = @{}
    )

    # Keep caches inside the repository so Windows hosts do not depend on a writable profile cache.
    New-Item -ItemType Directory -Force -Path $GoCache, $GoModCache | Out-Null

    $effectiveEnvironment = @{
        GOCACHE     = $GoCache
        GOMODCACHE  = $GoModCache
        CGO_ENABLED = "0"
        GOOS        = $null
        GOARCH      = $null
    }

    foreach ($entry in $Environment.GetEnumerator()) {
        $effectiveEnvironment[$entry.Key] = [string]$entry.Value
    }

    $savedEnvironment = @{}
    foreach ($name in $effectiveEnvironment.Keys) {
        $savedEnvironment[$name] = Get-ProcessEnvironmentValue -Name $name
        Set-ProcessEnvironmentValue -Name $name -Value $effectiveEnvironment[$name]
    }

    try {
        & go @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "go $($Arguments -join ' ') failed with exit code $LASTEXITCODE."
        }
    }
    finally {
        foreach ($name in $effectiveEnvironment.Keys) {
            Set-ProcessEnvironmentValue -Name $name -Value $savedEnvironment[$name]
        }
    }
}

function Get-HostExecutableExtension {
    if ($env:OS -eq "Windows_NT") {
        return ".exe"
    }

    return ""
}

function Invoke-HostBuild {
    New-Item -ItemType Directory -Force -Path $Bin | Out-Null
    $output = Join-Path $Bin "$App$(Get-HostExecutableExtension)"
    Invoke-Go -Arguments @("build", "-trimpath", "-ldflags", $script:LinkerFlags, "-o", $output, $Source)
}

function Invoke-WindowsBuild {
    param([Parameter(Mandatory)][string]$Arch)

    New-Item -ItemType Directory -Force -Path $Dist | Out-Null
    $output = Join-Path $Dist "$App-windows-$Arch.exe"
    Invoke-Go -Arguments @("build", "-trimpath", "-ldflags", $script:LinkerFlags, "-o", $output, $Source) -Environment @{
        GOOS   = "windows"
        GOARCH = $Arch
    }
}

function Invoke-FormattingCheck {
    $files = & gofmt -l .
    if ($LASTEXITCODE -ne 0) {
        throw "gofmt -l . failed with exit code $LASTEXITCODE."
    }

    if ($files) {
        Write-Host "These files need gofmt:"
        $files | ForEach-Object { Write-Host $_ }
        throw "Formatting check failed."
    }
}

function Invoke-Clean {
    foreach ($path in @($Bin, $Dist, $GoCache, $GoModCache)) {
        if (Test-Path $path) {
            Remove-Item -Recurse -Force $path
        }
    }
}

function Show-Help {
    @"
Usage:
  .\build.ps1 <task> [<task> ...]

Common tasks:
  test                  Run go test ./...
  build                 Build the host binary into bin/
  build-windows         Build dist/nssmr-windows-amd64.exe and arm64.exe
  build-windows-amd64   Build dist/nssmr-windows-amd64.exe
  build-windows-arm64   Build dist/nssmr-windows-arm64.exe
  vet                   Run go vet ./...
  lint                  Check gofmt output and run go vet
  fmt                   Run go fmt ./...
  clean                 Remove build outputs and local caches
  help                  Show this help

Examples:
  .\build.ps1 test build
  .\build.ps1 build-windows
  .\build.ps1 build-windows -WindowsArch amd64
"@ | Write-Output
}

foreach ($currentTask in $Task) {
    switch ($currentTask) {
        "help" {
            Show-Help
        }
        "test" {
            Invoke-Go -Arguments @("test", $Packages)
        }
        "build" {
            Invoke-HostBuild
        }
        "build-windows" {
            foreach ($arch in $WindowsArch) {
                Invoke-WindowsBuild -Arch $arch
            }
        }
        "build-windows-amd64" {
            Invoke-WindowsBuild -Arch "amd64"
        }
        "build-windows-arm64" {
            Invoke-WindowsBuild -Arch "arm64"
        }
        "vet" {
            Invoke-Go -Arguments @("vet", $Packages)
        }
        "lint" {
            Invoke-FormattingCheck
            Invoke-Go -Arguments @("vet", $Packages)
        }
        "fmt" {
            Invoke-Go -Arguments @("fmt", $Packages)
        }
        "clean" {
            Invoke-Clean
        }
    }
}
