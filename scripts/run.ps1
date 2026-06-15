$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()

$Root = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $Root "backend"
$FrontendDir = Join-Path $Root "frontend"
$LogDir = Join-Path $Root "logs"
$EnvFile = Join-Path $BackendDir ".env.local"

function Stop-Port($Port) {
  $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  foreach ($connection in $connections) {
    if ($connection.OwningProcess -and $connection.OwningProcess -ne 0) {
      Stop-Process -Id $connection.OwningProcess -Force -ErrorAction SilentlyContinue
    }
  }
}

function Load-Env($Path) {
  if (-not (Test-Path $Path)) {
    Write-Host "No backend/.env.local found. Defaults will be used." -ForegroundColor Yellow
    return
  }
  Get-Content -Encoding UTF8 $Path | ForEach-Object {
    if ($_ -match "^\s*([^#][^=]+)=(.*)$") {
      [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2], "Process")
    }
  }
}

New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

Write-Host "Stopping old services..." -ForegroundColor Cyan
Stop-Port 8080
Stop-Port 5173

Load-Env $EnvFile

if (-not (Test-Path (Join-Path $FrontendDir "node_modules"))) {
  Write-Host "Installing frontend dependencies..." -ForegroundColor Cyan
  Start-Process -FilePath "npm.cmd" -ArgumentList @("install") -WorkingDirectory $FrontendDir -Wait -WindowStyle Hidden
}

Write-Host "Starting backend http://localhost:8080 ..." -ForegroundColor Cyan
$backendOut = Join-Path $LogDir "backend.log"
$backendErr = Join-Path $LogDir "backend.err.log"
$backend = Start-Process -FilePath "go" -ArgumentList @("run", ".\cmd\server") -WorkingDirectory $BackendDir -PassThru -WindowStyle Hidden -RedirectStandardOutput $backendOut -RedirectStandardError $backendErr
$backend.Id | Set-Content -Path (Join-Path $BackendDir ".server.pid") -Encoding ASCII

Write-Host "Starting frontend http://localhost:5173 ..." -ForegroundColor Cyan
$frontendOut = Join-Path $LogDir "frontend.log"
$frontendErr = Join-Path $LogDir "frontend.err.log"
$frontend = Start-Process -FilePath "npm.cmd" -ArgumentList @("run", "dev", "--", "--host", "0.0.0.0") -WorkingDirectory $FrontendDir -PassThru -WindowStyle Hidden -RedirectStandardOutput $frontendOut -RedirectStandardError $frontendErr
$frontend.Id | Set-Content -Path (Join-Path $FrontendDir ".vite.pid") -Encoding ASCII

Start-Sleep -Seconds 2
Write-Host "Started." -ForegroundColor Green
Write-Host "Frontend: http://localhost:5173"
Write-Host "Backend: http://localhost:8080"
Write-Host "Backend log: $backendOut"
Write-Host "Frontend log: $frontendOut"
