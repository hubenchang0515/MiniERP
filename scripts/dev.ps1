param([string]$ApiTarget = 'http://127.0.0.1:8080')
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$backend = $null
$oldTarget = $env:VITE_API_TARGET
try {
    Push-Location (Join-Path $root 'frontend')
    try {
        npm ci; if ($LASTEXITCODE -ne 0) { throw '依赖安装失败' }
        npm run build; if ($LASTEXITCODE -ne 0) { throw '前端构建失败' }
        Copy-Item -Path 'dist/*' -Destination (Join-Path $root 'backend/internal/web/static') -Recurse -Force
    } finally { Pop-Location }
    Push-Location (Join-Path $root 'backend')
    try { go build -o (Join-Path $root 'backend/dev-minierp.exe') ./cmd/minierp; if ($LASTEXITCODE -ne 0) { throw '后端构建失败' } } finally { Pop-Location }
    $backend = Start-Process -FilePath (Join-Path $root 'backend/dev-minierp.exe') -WorkingDirectory $root -PassThru -WindowStyle Hidden
    $env:VITE_API_TARGET = $ApiTarget
    Push-Location (Join-Path $root 'frontend')
    try { npm run dev } finally { Pop-Location }
} finally { $env:VITE_API_TARGET = $oldTarget; if ($backend -and -not $backend.HasExited) { Stop-Process -Id $backend.Id } }
