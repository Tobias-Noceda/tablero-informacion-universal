#requires -Version 7
<#
.SYNOPSIS
    Run the Go test suite with coverage and produce a report.
.DESCRIPTION
    Runs `go test` over the backend module with a cross-package coverage
    profile, prints a per-function summary plus the total, and writes an
    HTML report. Pass -Open to launch the HTML in your browser.
.EXAMPLE
    ./coverage.ps1
    ./coverage.ps1 -Open
#>
param(
    [switch]$Open
)

$ErrorActionPreference = 'Stop'

$backend = Join-Path $PSScriptRoot 'backend'
$profilePath = Join-Path $backend 'coverage.out'
$htmlPath    = Join-Path $backend 'coverage.html'

Push-Location $backend
try {
    Write-Host '==> Running tests with coverage...' -ForegroundColor Cyan
    go test ./common/... -covermode=atomic "-coverprofile=$profilePath"
    if ($LASTEXITCODE -ne 0) {
        Write-Host 'Tests failed.' -ForegroundColor Red
        exit $LASTEXITCODE
    }

    Write-Host "`n==> Per-function coverage:" -ForegroundColor Cyan
    go tool cover -func "$profilePath"

    go tool cover -html "$profilePath" -o "$htmlPath"
    Write-Host "`n==> HTML report: $htmlPath" -ForegroundColor Green

    if ($Open) {
        Start-Process $htmlPath
    }
}
finally {
    Pop-Location
}
