# Usage: .\install-git-kanban.ps1 C:\path\to\git-kanban

param(
  [string]$Script
)

$Target = "git-kanban"
if (-not $Script -or -not (Test-Path $Script)) {
  Write-Host "Usage: .\install-git-kanban.ps1 C:\path\to\git-kanban"
  exit 1
}

$BinDir = "$env:USERPROFILE\bin"
if (-not (Test-Path $BinDir)) {
  New-Item -ItemType Directory -Path $BinDir | Out-Null
}

Copy-Item $Script "$BinDir\$Target"
Write-Host "Installed to $BinDir\$Target"

# Add to PATH if not present
if (-not ($env:PATH -like "*$BinDir*")) {
  [Environment]::SetEnvironmentVariable("PATH", "$env:PATH;$BinDir", [EnvironmentVariableTarget]::User)
  Write-Host "Added $BinDir to PATH. Restart your terminal to use git kanban."
}

Write-Host "You can now run: git kanban"
