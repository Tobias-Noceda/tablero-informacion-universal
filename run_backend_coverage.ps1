#requires -Version 7
<#
.SYNOPSIS
    Run the Go test suite with coverage and produce a report.
.DESCRIPTION
    Runs `go test` over the backend module (excluding test doubles and the live
    driver wrappers), prints the per-function breakdown followed by a per-package
    + total summary, and writes an HTML report. Pass -Open to launch the HTML in
    your browser.
.EXAMPLE
    ./run_backend_coverage.ps1
    ./run_backend_coverage.ps1 -Open
#>
param(
    [switch]$Open
)

$ErrorActionPreference = 'Stop'

$backend = Join-Path $PSScriptRoot 'backend'
$profilePath = Join-Path $backend 'coverage.out'
$htmlPath    = Join-Path $backend 'coverage.html'

$excludePattern = 'ports/(mongo|redis)|/mocks'

Push-Location $backend
try {
    $pkgs = go list ./common/... | Where-Object { $_ -notmatch $excludePattern }

    Write-Host '==> Running tests with coverage...' -ForegroundColor Cyan
    # -covermode=atomic records per-block hit counts (goroutine-safe; the cache
    # write in ExecutePostIt runs in a goroutine), surfacing weakly-hit branches.
    $testOut = go test $pkgs -covermode=atomic "-coverprofile=$profilePath" -cover
    $testOut | Write-Host
    if ($LASTEXITCODE -ne 0) {
        Write-Host 'Tests failed.' -ForegroundColor Red
        exit $LASTEXITCODE
    }

    $funcOut = go tool cover -func "$profilePath"

    Write-Host "`n==> Per-function coverage:" -ForegroundColor Cyan
    $funcOut | Write-Host

    Write-Host "`n==> Coverage summary" -ForegroundColor Cyan

    # Statement coverage per package (from `go test`).
    $stmt = @{}
    foreach ($line in $testOut) {
        if ($line -match 'coverage:\s+([\d.]+)%') {
            $pkg = ($line -split '\s+')[1] -replace '.*/common/', ''
            $stmt[$pkg] = [double]$Matches[1]
        }
    }

    # Function coverage per package: a function counts as covered if any of its
    # statements ran
    $fnTotal = @{}; $fnCovered = @{}; $allFn = 0; $allFnCov = 0
    foreach ($line in $funcOut) {
        if ($line -match '^total:') { continue }
        if ($line -match '^(\S+\.go):\d+:\s+\S+\s+([\d.]+)%') {
            $pkg = $Matches[1] -replace '.*/common/', '' -replace '/[^/]+\.go$', ''
            if (-not $fnTotal.ContainsKey($pkg)) { $fnTotal[$pkg] = 0; $fnCovered[$pkg] = 0 }
            $fnTotal[$pkg]++; $allFn++
            if (([double]$Matches[2]) -gt 0) { $fnCovered[$pkg]++; $allFnCov++ }
        }
    }

    Write-Host ("    {0,-22} {1,7} {2,7}" -f 'package', 'stmt', 'func')
    foreach ($pkg in ($stmt.Keys | Sort-Object)) {
        $s = $stmt[$pkg]
        $f = if ($fnTotal[$pkg]) { 100.0 * $fnCovered[$pkg] / $fnTotal[$pkg] } else { 0.0 }
        $color = if ($s -ge 80) { 'Green' } elseif ($s -ge 50) { 'Yellow' } else { 'Red' }
        Write-Host ("    {0,-22} {1,6:N1}% {2,6:N1}%" -f $pkg, $s, $f) -ForegroundColor $color
    }

    # Totals: statement from the profile, function from the aggregate counts.
    $totalLine = $funcOut | Select-String -Pattern '^total:'
    $totalStmt = if ($totalLine -and $totalLine.ToString() -match '([\d.]+)%') { [double]$Matches[1] } else { 0.0 }
    $totalFn = if ($allFn) { 100.0 * $allFnCov / $allFn } else { 0.0 }
    $color = if ($totalStmt -ge 80) { 'Green' } elseif ($totalStmt -ge 50) { 'Yellow' } else { 'Red' }
    Write-Host ("    {0,-22} {1,6:N1}% {2,6:N1}%" -f 'TOTAL', $totalStmt, $totalFn) -ForegroundColor $color

    go tool cover -html "$profilePath" -o "$htmlPath"
    Write-Host "`n==> HTML report: $htmlPath" -ForegroundColor Green

    if ($Open) {
        Start-Process $htmlPath
    }
}
finally {
    Pop-Location
}
