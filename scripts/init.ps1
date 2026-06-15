param(
  [switch]$Force,
  [string]$AdminEmail = "",
  [string]$AdminName = "",
  [string]$AdminPassword = "",
  [string]$HttpAddr = ""
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()

$Root = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $Root "backend"
$DataDir = Join-Path $BackendDir "data"
$LogDir = Join-Path $Root "logs"
$EnvFile = Join-Path $BackendDir ".env.local"
$DbFiles = @(
  (Join-Path $DataDir "bitapi.db"),
  (Join-Path $DataDir "bitapi.db-shm"),
  (Join-Path $DataDir "bitapi.db-wal")
)

function Read-Required($Prompt, $Default = "") {
  while ($true) {
    if ($Default) {
      $value = Read-Host "$Prompt [$Default]"
      if ([string]::IsNullOrWhiteSpace($value)) {
        $value = $Default
      }
    } else {
      $value = Read-Host $Prompt
    }
    if (-not [string]::IsNullOrWhiteSpace($value)) {
      return $value.Trim()
    }
    Write-Host "This value is required." -ForegroundColor Yellow
  }
}

function Read-PasswordText($Prompt) {
  while ($true) {
    $secure = Read-Host $Prompt -AsSecureString
    $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    try {
      $value = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
    } finally {
      [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
    }
    if (-not [string]::IsNullOrWhiteSpace($value)) {
      return $value
    }
    Write-Host "Password is required." -ForegroundColor Yellow
  }
}

function New-Secret($Bytes) {
  $buffer = [byte[]]::new($Bytes)
  $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
  try {
    $rng.GetBytes($buffer)
  } finally {
    $rng.Dispose()
  }
  return [Convert]::ToBase64String($buffer)
}

function Stop-Port($Port) {
  $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  foreach ($connection in $connections) {
    if ($connection.OwningProcess -and $connection.OwningProcess -ne 0) {
      Stop-Process -Id $connection.OwningProcess -Force -ErrorAction SilentlyContinue
    }
  }
}

Write-Host "BitAPI init" -ForegroundColor Cyan
Write-Host "This will stop local 8080/5173 services and remove backend/data/bitapi.db." -ForegroundColor Yellow
if (-not $Force) {
  $confirm = Read-Host "Type YES to continue"
  if ($confirm -ne "YES") {
    Write-Host "Canceled."
    exit 0
  }
}

if ([string]::IsNullOrWhiteSpace($AdminEmail)) {
  $AdminEmail = Read-Required "Admin email" "admin@bitapi.local"
}
if ([string]::IsNullOrWhiteSpace($AdminName)) {
  $AdminName = Read-Required "Admin display name" "BitAPI Admin"
}
if ([string]::IsNullOrWhiteSpace($AdminPassword)) {
  $AdminPassword = Read-PasswordText "Admin password"
}
if ([string]::IsNullOrWhiteSpace($HttpAddr)) {
  $HttpAddr = Read-Required "Backend listen address" ":8080"
}

Stop-Port 8080
Stop-Port 5173

New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null
foreach ($file in $DbFiles) {
  if (Test-Path $file) {
    Remove-Item -LiteralPath $file -Force
  }
}

$jwtSecret = New-Secret 48
$encryptionKey = New-Secret 32

$envContent = @"
BITAPI_APP_NAME=BitAPI
BITAPI_ENV=production
BITAPI_HTTP_ADDR=$HttpAddr
BITAPI_DATABASE_DSN=file:data/bitapi.db?_foreign_keys=on&_busy_timeout=5000
BITAPI_JWT_SECRET=$jwtSecret
BITAPI_ACCESS_TOKEN_TTL=30m
BITAPI_REFRESH_TOKEN_TTL=336h
BITAPI_CORS_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
BITAPI_BOOTSTRAP_EMAIL=$AdminEmail
BITAPI_BOOTSTRAP_PASSWORD=$AdminPassword
BITAPI_BOOTSTRAP_NAME=$AdminName
BITAPI_ENCRYPTION_KEY=$encryptionKey
BITAPI_DEFAULT_USER_BALANCE_MICROS=0
"@
[IO.File]::WriteAllText($EnvFile, $envContent, [Text.UTF8Encoding]::new($false))

Get-Content -Encoding UTF8 $EnvFile | ForEach-Object {
  if ($_ -match "^\s*([^#][^=]+)=(.*)$") {
    [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2], "Process")
  }
}

$initOut = Join-Path $LogDir "init-backend.log"
$initErr = Join-Path $LogDir "init-backend.err.log"
Push-Location $BackendDir
try {
  Write-Host "Creating tables and bootstrap admin..." -ForegroundColor Cyan
  $process = Start-Process -FilePath "go" -ArgumentList @("run", ".\cmd\server") -PassThru -WindowStyle Hidden -RedirectStandardOutput $initOut -RedirectStandardError $initErr
  $dbPath = Join-Path $DataDir "bitapi.db"
  $deadline = (Get-Date).AddSeconds(90)
  while ((Get-Date) -lt $deadline) {
    if (Test-Path $dbPath) {
      Start-Sleep -Seconds 2
      break
    }
    if ($process.HasExited) {
      throw "Backend exited before database was created. Check $initErr"
    }
    Start-Sleep -Seconds 1
  }
  if (-not (Test-Path $dbPath)) {
    throw "Database was not created in time. Check $initErr"
  }
  if (-not $process.HasExited) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  }
  Stop-Port 8080
} finally {
  Pop-Location
}

Write-Host "Init completed." -ForegroundColor Green
Write-Host "Admin email: $AdminEmail"
Write-Host "Config file: $EnvFile"
