param(
  [switch]$Force,
  [string]$AdminEmail = "",
  [string]$AdminName = "",
  [string]$AdminPassword = "",
  [string]$AppName = "",
  [string]$HttpAddr = ""
)

$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()

$Root = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $Root "backend"
$DataDir = Join-Path $BackendDir "data"
$LogDir = Join-Path $Root "logs"
$EnvFile = Join-Path $BackendDir ".env.local"
$BackendPort = 8091

function Read-Required {
  param(
    [string]$Prompt,
    [string]$Default = ""
  )
  while ($true) {
    if ([string]::IsNullOrWhiteSpace($Default)) {
      $value = Read-Host $Prompt
    } else {
      $value = Read-Host "$Prompt [$Default]"
      if ([string]::IsNullOrWhiteSpace($value)) {
        $value = $Default
      }
    }
    if (-not [string]::IsNullOrWhiteSpace($value)) {
      return $value.Trim()
    }
    Write-Host "该项不能为空，请重新输入。" -ForegroundColor Yellow
  }
}

function Read-PlainPassword {
  param([string]$Prompt)
  while ($true) {
    $value = Read-Host $Prompt
    if (-not [string]::IsNullOrWhiteSpace($value)) {
      return $value
    }
    Write-Host "密码不能为空，请重新输入。" -ForegroundColor Yellow
  }
}

function New-Secret {
  param([int]$Bytes)
  $buffer = New-Object byte[] $Bytes
  $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
  try {
    $rng.GetBytes($buffer)
  } finally {
    $rng.Dispose()
  }
  return [Convert]::ToBase64String($buffer)
}

function Stop-Port {
  param([int]$Port)
  $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  foreach ($connection in $connections) {
    if ($connection.OwningProcess -and $connection.OwningProcess -ne 0) {
      Stop-Process -Id $connection.OwningProcess -Force -ErrorAction SilentlyContinue
    }
  }
}

function Test-Healthy {
  param([string]$Url)
  try {
    Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 2 | Out-Null
    return $true
  } catch {
    return $false
  }
}

Write-Host "BitAPI 初始化向导" -ForegroundColor Cyan
Write-Host "注意：初始化会停止本机 $BackendPort 端口服务，并清空 backend\data\bitapi.db 数据库。" -ForegroundColor Yellow

if (-not $Force) {
  $confirm = Read-Host "请输入 YES 继续初始化，输入其它内容将取消"
  if ($confirm -ne "YES") {
    Write-Host "已取消初始化。"
    exit 0
  }
}

if ([string]::IsNullOrWhiteSpace($AppName)) {
  $AppName = Read-Required "请输入站点/产品名称" "BitAPI"
}
if ([string]::IsNullOrWhiteSpace($AdminEmail)) {
  $AdminEmail = Read-Required "请输入管理员账号邮箱" "admin@bitapi.local"
}
if ([string]::IsNullOrWhiteSpace($AdminName)) {
  $AdminName = Read-Required "请输入管理员名称" "$AppName 管理员"
}
if ([string]::IsNullOrWhiteSpace($AdminPassword)) {
  $AdminPassword = Read-PlainPassword "请输入管理员密码，输入过程会直接显示，请认真核对"
}
if ([string]::IsNullOrWhiteSpace($HttpAddr)) {
  $HttpAddr = Read-Required "请输入服务监听地址" ":$BackendPort"
}

Stop-Port 8080
Stop-Port 5173
Stop-Port 5181
Stop-Port $BackendPort

New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

Get-ChildItem -LiteralPath $DataDir -Filter "bitapi.db*" -ErrorAction SilentlyContinue | Remove-Item -Force

$jwtSecret = New-Secret 48
$encryptionKey = New-Secret 32

$envLines = @()
$envLines += "BITAPI_APP_NAME=$AppName"
$envLines += "BITAPI_ENV=production"
$envLines += "BITAPI_HTTP_ADDR=$HttpAddr"
$envLines += "BITAPI_DATABASE_DSN=file:data/bitapi.db?_foreign_keys=on&_busy_timeout=5000"
$envLines += "BITAPI_JWT_SECRET=$jwtSecret"
$envLines += "BITAPI_ACCESS_TOKEN_TTL=30m"
$envLines += "BITAPI_REFRESH_TOKEN_TTL=336h"
$envLines += "BITAPI_CORS_ORIGINS=http://localhost:$BackendPort,http://127.0.0.1:$BackendPort,https://www.bitit.cn,https://bitit.cn,https://demo.bitit.cn,https://*.bitit.cn"
$envLines += "BITAPI_BOOTSTRAP_EMAIL=$AdminEmail"
$envLines += "BITAPI_BOOTSTRAP_PASSWORD=$AdminPassword"
$envLines += "BITAPI_BOOTSTRAP_NAME=$AdminName"
$envLines += "BITAPI_ENCRYPTION_KEY=$encryptionKey"
$envLines += "BITAPI_DEFAULT_USER_BALANCE_MICROS=0"

$envContent = [string]::Join([Environment]::NewLine, $envLines) + [Environment]::NewLine
$utf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
[System.IO.File]::WriteAllText($EnvFile, $envContent, $utf8NoBom)

Get-Content -Encoding UTF8 $EnvFile | ForEach-Object {
  if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2], "Process")
  }
}

$initOut = Join-Path $LogDir "init-backend.log"
$initErr = Join-Path $LogDir "init-backend.err.log"
$process = $null

Push-Location $BackendDir
try {
  Write-Host "正在创建数据库表和管理员账号..." -ForegroundColor Cyan
  $process = Start-Process -FilePath "go" -ArgumentList @("run", ".\cmd\server") -PassThru -WindowStyle Hidden -RedirectStandardOutput $initOut -RedirectStandardError $initErr
  $dbPath = Join-Path $DataDir "bitapi.db"
  $deadline = (Get-Date).AddSeconds(90)

  while ((Get-Date) -lt $deadline) {
    if ((Test-Path $dbPath) -and (Test-Healthy "http://127.0.0.1:$BackendPort/health")) {
      break
    }
    if ($process.HasExited) {
      throw "后端服务提前退出，数据库没有创建成功。请查看：$initErr"
    }
    Start-Sleep -Seconds 1
  }

  if (-not (Test-Path $dbPath)) {
    throw "数据库创建超时。请查看：$initErr"
  }
  if (-not (Test-Healthy "http://127.0.0.1:$BackendPort/health")) {
    throw "后端健康检查超时。请查看：$initErr"
  }
} finally {
  if ($process -and -not $process.HasExited) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  }
  Stop-Port $BackendPort
  Pop-Location
}

Write-Host "初始化完成。" -ForegroundColor Green
Write-Host "管理员账号邮箱：$AdminEmail"
Write-Host "管理员密码：初始化时输入的密码"
Write-Host "管理员名称：$AdminName"
Write-Host "本地登录地址：http://localhost:$BackendPort/auth/login"
Write-Host "配置文件位置：$EnvFile"
Write-Host "提示：熟悉命令行的用户也可以使用 -AdminEmail -AdminName -AdminPassword -HttpAddr 参数。"
