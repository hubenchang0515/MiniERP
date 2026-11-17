param([Alias('h')][switch]$Help)

$ErrorActionPreference = 'Stop'
if ($Help) {
    Write-Output '用法: .\scripts\install-deps.ps1'
    Write-Output '安装前端依赖（npm ci）并下载后端 Go 模块（go mod download）。首次使用或依赖变化后运行。'
    return
}
foreach ($toolName in @('npm', 'go')) {
    if (-not (Get-Command $toolName -ErrorAction SilentlyContinue)) {
        throw "未找到 $toolName，请先安装 Node.js/npm 和 Go，并将它们加入 PATH。"
    }
}
$root = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $root 'frontend')
try {
    npm ci
    if ($LASTEXITCODE -ne 0) { throw '前端依赖安装失败' }
} finally { Pop-Location }
Push-Location (Join-Path $root 'backend')
try {
    go mod download
    if ($LASTEXITCODE -ne 0) { throw '后端依赖下载失败' }
} finally { Pop-Location }
Write-Host '依赖安装完成。'
