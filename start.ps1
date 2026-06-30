# AI-Drama One Click Start / Restart
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  AI Drama - Start" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ServerDir = Join-Path $ScriptDir "server"
$WebDir    = Join-Path $ScriptDir "web"

# ============================================================
# Helper: recursively kill a process and all its children
# ============================================================
function Kill-ProcessTree {
    param([int]$Pid)
    if ($Pid -le 0) { return }
    # kill children first
    $children = Get-CimInstance Win32_Process | Where-Object { $_.ParentProcessId -eq $Pid }
    foreach ($child in $children) {
        Kill-ProcessTree -Pid $child.ProcessId
    }
    Stop-Process -Id $Pid -Force -ErrorAction SilentlyContinue
}

# ============================================================
# [1/4] Kill all old processes (process tree, not just listener)
# ============================================================
Write-Host "[1/4] Cleanup old processes..." -ForegroundColor Yellow

# --- Backend ---
$backendProc = Get-Process -Name "spirit-fruit-server" -ErrorAction SilentlyContinue
if ($backendProc) {
    $backendProc | ForEach-Object { Kill-ProcessTree -Pid $_.Id }
    Write-Host "  Backend killed" -ForegroundColor Green
} else {
    Write-Host "  No old backend" -ForegroundColor Gray
}

# --- Frontend (Vite on port 3002) ---
$conns3002 = Get-NetTCPConnection -LocalPort 3002 -ErrorAction SilentlyContinue | Where-Object { $_.State -eq 'Listen' }
foreach ($c in $conns3002) {
    Kill-ProcessTree -Pid $c.OwningProcess
    Write-Host "  Frontend killed (PID: $($c.OwningProcess))" -ForegroundColor Green
}

# --- Ensure port 3002 is free ---
$retry = 0
while ($retry -lt 10) {
    $still = Get-NetTCPConnection -LocalPort 3002 -ErrorAction SilentlyContinue | Where-Object { $_.State -eq 'Listen' }
    if (-not $still) { break }
    Start-Sleep -Seconds 1
    $retry++
}
if ($retry -ge 10) {
    Write-Host "  WARNING: Port 3002 still occupied" -ForegroundColor Red
}

Write-Host ""

# ============================================================
# [2/5] Start Redis (if not running)
# ============================================================
Write-Host "[2/5] Checking Redis..." -ForegroundColor Yellow
$redisRunning = Get-NetTCPConnection -LocalPort 6379 -ErrorAction SilentlyContinue | Where-Object { $_.State -eq 'Listen' }
if (-not $redisRunning) {
    $redisDir = "D:\software\Redis-x64-5.0.14.1"
    if (Test-Path "$redisDir\redis-server.exe") {
        Start-Process -FilePath "$redisDir\redis-server.exe" -ArgumentList "$redisDir\redis.windows-service.conf" -WindowStyle Hidden
        Write-Host "  Redis started" -ForegroundColor Green
    } else {
        Write-Host "  Redis not found at $redisDir, skip" -ForegroundColor Yellow
    }
} else {
    Write-Host "  Redis already running" -ForegroundColor Gray
}

# ============================================================
# [3/5] Build backend
# ============================================================
Write-Host "[3/5] Building backend..." -ForegroundColor Yellow
Push-Location $ServerDir
$buildResult = go build -o spirit-fruit-server.exe .
if ($LASTEXITCODE -ne 0) {
    Write-Host "  BUILD FAILED!" -ForegroundColor Red
    Pop-Location
    Read-Host "Press Enter to exit"
    exit 1
}
Pop-Location
Write-Host "  Build OK" -ForegroundColor Green

# ============================================================
# [4/5] Start backend
# ============================================================
Write-Host "[3/4] Starting backend..." -ForegroundColor Yellow
Start-Process -FilePath "$ServerDir\spirit-fruit-server.exe" -ArgumentList "serve" -WorkingDirectory $ServerDir -WindowStyle Normal

# ============================================================
# [5/5] Start frontend
# ============================================================
Write-Host "[4/4] Starting frontend..." -ForegroundColor Yellow
Start-Process -FilePath "cmd.exe" -ArgumentList "/c pnpm run dev" -WorkingDirectory $WebDir -WindowStyle Normal

Write-Host ""
Write-Host "Waiting for services to start..." -ForegroundColor Cyan
Start-Sleep -Seconds 5

# ============================================================
# Open browser only if not already open
# ============================================================
$alreadyOpen = $false
$edgeProcs = Get-Process -Name "msedge" -ErrorAction SilentlyContinue
foreach ($ep in $edgeProcs) {
    if ($ep.MainWindowTitle -match "AI|Drama|3002|localhost") {
        $alreadyOpen = $true
        break
    }
}
if (-not $alreadyOpen) {
    Start-Process "http://localhost:3002"
    Write-Host ""
    Write-Host "Done! Browser opened at http://localhost:3002" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "Done! Refresh your browser (Ctrl+Shift+R)" -ForegroundColor Green
}

Write-Host ""
Read-Host "Press Enter to exit"
