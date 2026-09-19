#requires -Version 7
<#
.SYNOPSIS
    Run the Go integration tests (`//go:build integration`).
.DESCRIPTION
    Starts the mock OAuth2 provider from docker-compose (profile `integration`,
    listening on localhost:8899), waits for it, and runs the tagged tests. The
    Duende test also needs internet access; filter it out with -Run if offline.
.EXAMPLE
    ./run_backend_integration.ps1
    ./run_backend_integration.ps1 -Run TestLiveAuthorizationCode
    ./run_backend_integration.ps1 -Keep      # leave the mock running afterwards
#>
param(
    [string]$Run,
    [switch]$Keep
)

$ErrorActionPreference = 'Stop'

$backend = Join-Path $PSScriptRoot 'backend'
$mockUrl = 'http://localhost:8899/default/.well-known/openid-configuration'

Push-Location $PSScriptRoot
try {
    Write-Host '==> Starting mock OAuth2 provider...' -ForegroundColor Cyan
    docker compose --profile integration up -d mock-oauth2
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

    $ready = $false
    for ($i = 0; $i -lt 30 -and -not $ready; $i++) {
        try {
            Invoke-WebRequest -Uri $mockUrl -UseBasicParsing -TimeoutSec 2 | Out-Null
            $ready = $true
        } catch {
            Start-Sleep -Seconds 1
        }
    }
    if (-not $ready) {
        Write-Host "Mock OAuth2 provider did not come up at $mockUrl" -ForegroundColor Red
        exit 1
    }

    Push-Location $backend
    try {
        Write-Host '==> Running integration tests...' -ForegroundColor Cyan
        $goArgs = @('test', '-tags', 'integration', '-v')
        if ($Run) { $goArgs += @('-run', $Run) }
        $goArgs += './common/services/secrets/'
        & go @goArgs
        $exit = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
}
finally {
    if (-not $Keep) {
        Write-Host '==> Stopping mock OAuth2 provider...' -ForegroundColor Cyan
        docker compose --profile integration stop mock-oauth2 | Out-Null
    }
    Pop-Location
}

exit $exit
