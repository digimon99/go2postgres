$ErrorActionPreference = "Stop"

Write-Host ">>> Starting Build Process..." -ForegroundColor Cyan

# --- 1. Version Management ---
$VersionFile = "version.txt"
$BuildNumFile = "build_number.txt"

# Read current version
if (Test-Path $VersionFile) {
    $Version = (Get-Content $VersionFile -Raw).Trim()
} else {
    $Version = "1.0"
    Set-Content -Path $VersionFile -Value $Version -NoNewline
}

# Read and increment build number
if (Test-Path $BuildNumFile) {
    $BuildNum = [int](Get-Content $BuildNumFile -Raw).Trim()
} else {
    $BuildNum = 0
}
$BuildNum++
Set-Content -Path $BuildNumFile -Value $BuildNum -NoNewline

$BuildTime = Get-Date -Format "yyyy-MM-ddTHH:mm:ss"
$LdFlags = "-s -w -X main.Version=$Version -X main.BuildTime=$BuildTime -X main.BuildNum=$BuildNum"

Write-Host ">>> Version: $Version (Build #$BuildNum)" -ForegroundColor Magenta

# --- 2. Build Frontend ---
Write-Host ">>> Building Frontend (React)..." -ForegroundColor Cyan
Push-Location -Path "web"
try {
    # Install dependencies if needed
    if (-not (Test-Path "node_modules")) {
        Write-Host ">>> Installing npm dependencies..."
        npm install
        if ($LASTEXITCODE -ne 0) {
            Write-Error "npm install failed"
            exit 1
        }
    }
    
    # Build frontend
    npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Frontend build failed"
        exit 1
    }
    Write-Host ">>> Frontend Build Success" -ForegroundColor Green
} finally {
    Pop-Location
}

# --- 3. Build Go Binaries ---

# Windows
Write-Host ">>> Building Windows Binary (go2postgres.exe)..."
$Env:GOOS = "windows"; $Env:GOARCH = "amd64"; $Env:CGO_ENABLED = "0"
go build -o go2postgres.exe -ldflags "$LdFlags" ./cmd/go2postgres
if ($LASTEXITCODE -eq 0) { 
    Write-Host ">>> Windows Build Success" -ForegroundColor Green 
} else {
    Write-Error "Windows build failed"
    exit 1
}

# Linux
Write-Host ">>> Building Linux Binary (go2postgres-linux-new)..."
$Env:GOOS = "linux"; $Env:GOARCH = "amd64"; $Env:CGO_ENABLED = "0"
go build -o go2postgres-linux-new -ldflags "$LdFlags" ./cmd/go2postgres
if ($LASTEXITCODE -eq 0) { 
    Write-Host ">>> Linux Build Success" -ForegroundColor Green 
} else {
    Write-Error "Linux build failed"
    exit 1
}

# Show file sizes
Write-Host ""
Write-Host ">>> Build Artifacts:" -ForegroundColor Cyan
Get-ChildItem -Path "." -Filter "go2postgres*" | Where-Object { -not $_.PSIsContainer } | Format-Table Name, @{Label="Size (MB)"; Expression={[math]::Round($_.Length/1MB, 2)}}

Write-Host ">>> Build Complete! Version $Version Build #$BuildNum" -ForegroundColor Cyan
