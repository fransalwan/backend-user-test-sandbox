@echo off
setlocal
echo =========================================================================
echo   Fintech Hiring Gauntlet - Cloudflare Tunnel Starter
echo =========================================================================
echo.
echo Pastikan aplikasi backend sudah aktif di http://localhost:8080
echo (Bisa via sandbox.exe atau via docker compose up)
echo.

where cloudflared >nul 2>nul
if %ERRORLEVEL% equ 0 (
    echo [OK] cloudflared ditemukan! Membuka Cloudflare Tunnel ke port 8080...
    echo URL publik HTTPS akan muncul di bawah ini (contoh: https://xxxx.trycloudflare.com)
    echo Tekan Ctrl+C untuk menghentikan siaran publik.
    echo.
    cloudflared tunnel --url http://localhost:8080
) else (
    echo [INFO] cloudflared CLI belum terpasang di PATH sistem.
    echo.
    echo Anda dapat mengunduh cloudflared.exe resmi (gratis tanpa daftar akun):
    echo   1. Buka browser: https://github.com/cloudflare/cloudflared/releases/latest
    echo   2. Unduh file: cloudflared-windows-amd64.exe
    echo   3. Rename menjadi: cloudflared.exe dan letakkan di folder ini atau folder C:\Windows\System32
    echo   4. Jalankan ulang script ini, atau ketik:
    echo        cloudflared.exe tunnel --url http://localhost:8080
    echo.
    echo Atau via package manager Windows (winget):
    echo   winget install --id Cloudflare.cloudflared
    echo.
    pause
)
