param(
  [string]$ProjectRoot = "D:\Projeler\aurapanel"
)

$core = Join-Path $ProjectRoot "core"
$gateway = Join-Path $ProjectRoot "api-gateway"
$frontend = Join-Path $ProjectRoot "frontend"
$cargoExe = Join-Path $env:USERPROFILE ".cargo\bin\cargo.exe"
$coreExe = Join-Path $core "target\debug\aurapanel-core.exe"

goExe = (Get-Command go -ErrorAction SilentlyContinue).Source
$gatewayExe = Join-Path $gateway "apigw.exe"

Write-Host "Starting AuraPanel Core on :8000 ..."
if (Test-Path $cargoExe) {
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "$env:AURAPANEL_DEV_SIMULATION='1'; Set-Location '$core'; & '$cargoExe' run"
} elseif (Test-Path $coreExe) {
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "$env:AURAPANEL_DEV_SIMULATION='1'; Set-Location '$core'; & '$coreExe'"
} else {
  Write-Host "ERROR: Rust cargo not found and compiled backend not found at $coreExe"
  exit 1
}

Start-Sleep -Seconds 2

Write-Host "Starting API Gateway on :8080 ..."
if ($goExe) {
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "$env:AURAPANEL_DEV_SIMULATION='1'; Set-Location '$gateway'; & '$goExe' run ."
} elseif (Test-Path $gatewayExe) {
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "$env:AURAPANEL_DEV_SIMULATION='1'; Set-Location '$gateway'; & '$gatewayExe'"
} else {
  Write-Host "WARNING: Go runtime/gateway binary not found. Frontend may fail auth without gateway."
}

Start-Sleep -Seconds 2

Write-Host "Starting Frontend on :5173 ..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$frontend'; npm run dev"

Write-Host "Done. Core: http://127.0.0.1:8000  Gateway: http://127.0.0.1:8080  Frontend: http://127.0.0.1:5173"
