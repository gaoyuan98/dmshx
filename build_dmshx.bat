@echo off
echo Building dmshx for multiple platforms...

REM Set version info
set VERSION=1.0.0
set BUILD_TIME=%date% %time%
set AUTHOR=gaoyuan
set BUILD_DATE=20250617

REM Define optimization flags
set OPT_FLAGS=-s -w
set LDFLAGS=-ldflags "%OPT_FLAGS% -X dmshx/pkg.Version=%VERSION% -X 'dmshx/pkg.BuildTime=%BUILD_TIME%' -X dmshx/pkg.Author=%AUTHOR% -X dmshx/pkg.BuildDate=%BUILD_DATE%"

echo Building for Linux x86_64...
set GOOS=linux
set GOARCH=amd64
go build %LDFLAGS% -trimpath -o dmshx-linux-x86_64 ./cmd/dmshx

echo Building for Linux ARM64...
set GOOS=linux
set GOARCH=arm64
go build %LDFLAGS% -trimpath -o dmshx-linux-arm64 ./cmd/dmshx

echo Building for Windows x86_64...
set GOOS=windows
set GOARCH=amd64
go build %LDFLAGS% -trimpath -o dmshx-windows-x86_64.exe ./cmd/dmshx

echo Build completed successfully!
echo.
echo The following binaries have been created:
echo - dmshx-linux-x86_64 (Linux x86_64)
echo - dmshx-linux-arm64 (Linux ARM64)
echo - dmshx-windows-x86_64.exe (Windows x86_64) 