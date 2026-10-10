# =========================================================================
# Fintech Hiring Gauntlet - Cloudflare Tunnel Starter (PowerShell)
# Membuka port 8080 lokal ke domain publik HTTPS gratis via Cloudflare Tunnel
# =========================================================================

Write-Host "=========================================================================" -ForegroundColor Cyan
Write-Host "  Fintech Hiring Gauntlet - Cloudflare Tunnel Starter" -ForegroundColor Green
Write-Host "=========================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Memeriksa ketersediaan cloudflared CLI..." -ForegroundColor Yellow

$cloudflaredCmd = Get-Command cloudflared -ErrorAction SilentlyContinue

if ($cloudflaredCmd) {
    Write-Host "[OK] cloudflared ditemukan! Membuka tunnel ke http://localhost:8080..." -ForegroundColor Green
    Write-Host "Tautan publik HTTPS (contoh: https://xxxx.trycloudflare.com) akan digenerate otomatis." -ForegroundColor Yellow
    Write-Host "Tekan Ctrl+C untuk menghentikan tunnel." -ForegroundColor Gray
    Write-Host ""
    & cloudflared tunnel --url http://localhost:8080
} else {
    Write-Host "[INFO] cloudflared CLI belum terdeteksi di PATH sistem." -ForegroundColor Red
    Write-Host ""
    Write-Host "Cara termudah mengaktifkannya:" -ForegroundColor White
    Write-Host "  Opsi 1 (Otomatis via Winget):" -ForegroundColor Cyan
    Write-Host "    winget install --id Cloudflare.cloudflared" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Opsi 2 (Unduh Portable EXE):" -ForegroundColor Cyan
    Write-Host "    1. Buka: https://github.com/cloudflare/cloudflared/releases/latest" -ForegroundColor Gray
    Write-Host "    2. Download: cloudflared-windows-amd64.exe" -ForegroundColor Gray
    Write-Host "    3. Simpan sebagai 'cloudflared.exe' di folder ini dan jalankan:" -ForegroundColor Gray
    Write-Host "       .\cloudflared.exe tunnel --url http://localhost:8080" -ForegroundColor Yellow
    Write-Host ""
}

