# Build script for file-service (PowerShell)

Write-Host "Building file-service..." -ForegroundColor Green

# Clean previous builds
Remove-Item -Path "file-service.exe" -ErrorAction SilentlyContinue

# Build
go build -o file-service.exe .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build complete: file-service.exe" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
