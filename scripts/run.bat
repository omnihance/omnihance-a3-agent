@echo off

REM Set the input name for the server
set SERVER_NAME=%1
if "%SERVER_NAME%"=="" set SERVER_NAME=omnihance-a3-agent

REM Build UI first
echo Building UI...
cd cmd\%SERVER_NAME%\%SERVER_NAME%-ui
call pnpm run build
if errorlevel 1 (
    echo UI build failed!
    exit /b 1
)
cd ..\..\..

REM Run the Go application
go run .\cmd\%SERVER_NAME%\main.go
