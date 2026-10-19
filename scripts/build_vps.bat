@echo off
echo ========================================================
echo  Compilando Farm Ricochet Bot para Linux x86_64 (VPS)
echo ========================================================
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o farmbot-linux .
if %ERRORLEVEL% EQU 0 (
    echo.
    echo [SUCESSO] Binario 'farmbot-linux' gerado com sucesso!
) else (
    echo.
    echo [ERRO] Falha ao compilar.
)
pause
