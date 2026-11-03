$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
Set-Location $projectRoot
python scripts/init-env.py
if ($LASTEXITCODE -ne 0) { throw 'Environment setup failed' }
Get-Content -LiteralPath (Join-Path $projectRoot '.env') | ForEach-Object {
    if ($_ -match '^([A-Z_]+)=(.*)$') { [Environment]::SetEnvironmentVariable($Matches[1], $Matches[2], 'Process') }
}
$env:DRIFTLENS_DATA_DIR = Join-Path $projectRoot '.data'
$env:DRIFTLENS_REPOSITORY_ROOT = Join-Path $projectRoot 'demo'
$env:DRIFTLENS_CREDENTIAL_ROOT = Join-Path $projectRoot 'credentials'
$env:DRIFTLENS_FRONTEND_DIR = Join-Path $projectRoot 'frontend/dist'
Push-Location frontend
npm ci
if ($LASTEXITCODE -ne 0) { throw 'npm ci failed' }
npm run build
if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed' }
Pop-Location
Write-Host 'Open http://127.0.0.1:8080. The API token is in .env.'
Set-Location backend
go run ./cmd/server
