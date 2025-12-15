@echo off

REM Set the input name for the server and strip quotes
set SERVER_NAME=%1
set SERVER_NAME=%SERVER_NAME:"=%
if "%SERVER_NAME%"=="" set SERVER_NAME=omnihance-a3-agent

REM Set version (default to dev if not provided)
set VERSION=%2
if "%VERSION%"=="" set VERSION=dev
set VERSION=%VERSION:"=%

REM Set the Go environment variables for building for Linux (64-bit)
set GOARCH=amd64
set GOOS=linux

REM Build for Linux
echo Building %SERVER_NAME% for Linux (version: %VERSION%)...
go build -ldflags="-w -s -X main.version=%VERSION%" -o bin\%SERVER_NAME%\%SERVER_NAME% cmd\%SERVER_NAME%\main.go

REM Reset Go environment variables to their defaults
set GOARCH=
set GOOS=

REM Build for Windows
echo Building %SERVER_NAME% for Windows (version: %VERSION%)...
go build -ldflags="-w -s -X main.version=%VERSION%" -o bin\%SERVER_NAME%\%SERVER_NAME%.exe cmd\%SERVER_NAME%\main.go

echo %SERVER_NAME% build complete!
