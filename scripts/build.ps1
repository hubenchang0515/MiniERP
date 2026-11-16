param(
    [string]$Target,
    [Alias('h')][switch]$Help
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
function Check-Exit { if ($LASTEXITCODE -ne 0) { throw "构建失败，退出码 $LASTEXITCODE" } }

if ($Help) {
    Write-Output '用法: .\scripts\build.ps1 [-Target <系统/架构>]'
    Write-Output '默认仅构建当前主机平台。例如: -Target linux/amd64 或 -Target darwin/arm64'
    return
}
if (-not $PSBoundParameters.ContainsKey('Target')) {
    $hostPlatform = @(go env GOHOSTOS GOHOSTARCH)
    Check-Exit
    $Target = $hostPlatform -join '/'
}
$supportedTargets = @(go tool dist list)
Check-Exit
if ($Target -notmatch '^[a-z0-9]+/[a-z0-9]+$' -or $Target -notin $supportedTargets) {
    throw "不支持的构建目标: '$Target'。请使用系统/架构格式；可通过 go tool dist list 查看有效目标。"
}
$targetOS, $targetArch = $Target.Split('/')
$extension = if ($targetOS -eq 'windows') { '.exe' } else { '' }
Write-Host "构建目标: $Target"

Push-Location $root
$oldGOOS = $env:GOOS; $oldGOARCH = $env:GOARCH; $oldCGO = $env:CGO_ENABLED
try {
    Push-Location frontend
    try { npm ci; Check-Exit; npm run build; Check-Exit } finally { Pop-Location }
    $static = Join-Path $root 'backend/internal/web/static'
    # 仅清理已确认位于本项目中的生成目录，保留编译占位文件。
    $resolved = [IO.Path]::GetFullPath($static)
    if ($resolved -ne [IO.Path]::GetFullPath((Join-Path $root 'backend/internal/web/static'))) { throw '静态目录不正确' }
    Get-ChildItem -LiteralPath $resolved -Force | Where-Object { $_.Name -notin @('.keep', 'placeholder.html') } | ForEach-Object { Remove-Item -LiteralPath $_.FullName -Recurse -Force }
    Copy-Item -Path (Join-Path $root 'frontend/dist/*') -Destination $static -Recurse -Force
    New-Item -ItemType Directory -Force release | Out-Null
    Push-Location backend
    try {
        $env:CGO_ENABLED = '0'
        $env:GOOS = $targetOS; $env:GOARCH = $targetArch
        $output = Join-Path $root "release/minierp-$targetOS-$targetArch$extension"
        go build -trimpath -ldflags '-s -w' -o $output ./cmd/minierp; Check-Exit
        Write-Host "已构建 $output"
    } finally { Pop-Location }
} finally { $env:GOOS = $oldGOOS; $env:GOARCH = $oldGOARCH; $env:CGO_ENABLED = $oldCGO; Pop-Location }
