param([string]$ApiTarget = 'http://127.0.0.1:8080')
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$backend = $null
$oldTarget = $env:VITE_API_TARGET
$oldProxy = $env:GOPROXY
try {
    if (-not (Test-Path -LiteralPath (Join-Path $root 'frontend/node_modules') -PathType Container)) {
        throw '缺少前端依赖，请先运行 .\scripts\install-deps.ps1。'
    }
    Push-Location (Join-Path $root 'frontend')
    try {
        npm run build; if ($LASTEXITCODE -ne 0) { throw '前端构建失败' }
        Copy-Item -Path 'dist/*' -Destination (Join-Path $root 'backend/internal/web/static') -Recurse -Force
    } finally { Pop-Location }
    Push-Location (Join-Path $root 'backend')
    try {
        $env:GOPROXY = 'off'
        go build -mod=readonly -o (Join-Path $root 'backend/dev-minioa.exe') ./cmd/minioa
        if ($LASTEXITCODE -ne 0) { throw '后端构建失败；如缺少依赖，请先运行 .\scripts\install-deps.ps1。' }
    } finally { $env:GOPROXY = $oldProxy; Pop-Location }
    $backend = Start-Process -FilePath (Join-Path $root 'backend/dev-minioa.exe') -WorkingDirectory $root -PassThru -WindowStyle Hidden
    $env:VITE_API_TARGET = $ApiTarget
    Push-Location (Join-Path $root 'frontend')
    try { npm run dev } finally { Pop-Location }
} finally { $env:VITE_API_TARGET = $oldTarget; if ($backend -and -not $backend.HasExited) { Stop-Process -Id $backend.Id } }
