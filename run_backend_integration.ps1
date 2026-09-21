#requires -Version 7
<#
.SYNOPSIS
    Run the Go integration tests (`//go:build integration`).
.DESCRIPTION
    Brings up Mongo, Redis and the mock OAuth2 provider from docker-compose,
    points the tests at them through the environment (an ephemeral database
    name per run, dropped afterwards) and runs every tagged test in the module.
    TestLiveDuende also needs internet access; filter it out with -Run if offline.
.EXAMPLE
    ./run_backend_integration.ps1
    ./run_backend_integration.ps1 -Run TestEndToEnd
    ./run_backend_integration.ps1 -Keep      # leave the containers running afterwards
#>
param(
    [string]$Run,
    [switch]$Keep
)

$ErrorActionPreference = 'Stop'

$backend = Join-Path $PSScriptRoot 'backend'
$mockUrl = 'http://localhost:8899/default/.well-known/openid-configuration'

# Credentials come from .env, the same file docker compose reads.
$dotenv = @{}
Get-Content (Join-Path $PSScriptRoot '.env') | ForEach-Object {
    if ($_ -match '^\s*([^#=]+?)\s*=\s*(.*?)\s*$') { $dotenv[$Matches[1]] = $Matches[2] }
}
$mongoUser = $dotenv['MONGO_INITDB_ROOT_USERNAME']
$mongoPassword = $dotenv['MONGO_INITDB_ROOT_PASSWORD']
if (-not $mongoUser -or -not $mongoPassword) {
    Write-Host 'Set MONGO_INITDB_ROOT_USERNAME and MONGO_INITDB_ROOT_PASSWORD in .env' -ForegroundColor Red
    exit 1
}
$itDatabase = "it_$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"

$keyBytes = [byte[]]::new(32)
[System.Security.Cryptography.RandomNumberGenerator]::Fill($keyBytes)

$env:MONGODB_URI = "mongodb://${mongoUser}:${mongoPassword}@localhost:27017/?authSource=admin"
$env:MONGO_DATABASE = $itDatabase
$env:REDIS_URL = 'redis://localhost:6379/1'
$env:SECRETS_MASTER_KEYS = "1:$([Convert]::ToBase64String($keyBytes))"

function Invoke-Mongosh([string]$Script) {
    docker compose exec -T mongo mongosh --quiet -u $mongoUser -p $mongoPassword --authenticationDatabase admin --eval $Script
}

Push-Location $PSScriptRoot
try {
    Write-Host '==> Starting Mongo, Redis and the mock OAuth2 provider...' -ForegroundColor Cyan
    docker compose --profile integration up -d mongo redis mock-oauth2
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

    $ready = $false
    for ($i = 0; $i -lt 30 -and -not $ready; $i++) {
        Invoke-Mongosh 'db.runCommand({ping:1}).ok' 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) { $ready = $true } else { Start-Sleep -Seconds 1 }
    }

    Push-Location $backend
    try {
        Write-Host "==> Running integration tests against $itDatabase..." -ForegroundColor Cyan
        # Packages share one database, so they must not run concurrently.
        $goArgs = @('test', '-tags', 'integration', '-count=1', '-p', '1', '-v')
        if ($Run) { $goArgs += @('-run', $Run) }
        $goArgs += './...'
        & go @goArgs
        $exit = $LASTEXITCODE
    }
    finally {
        Pop-Location
    }
}
finally {
    Write-Host "==> Dropping database $itDatabase..." -ForegroundColor Cyan
    try { Invoke-Mongosh "db.getSiblingDB('$itDatabase').dropDatabase()" | Out-Null } catch {}
    if (-not $Keep) {
        Write-Host '==> Stopping the mock OAuth2 provider...' -ForegroundColor Cyan
        docker compose --profile integration stop mock-oauth2 | Out-Null
    }
    Pop-Location
}

exit $exit
