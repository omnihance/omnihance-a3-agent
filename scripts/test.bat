@echo off

go test ./... -coverprofile coverage.out
go tool cover -html=coverage.out -o coverage.html
for /f "tokens=3" %%i in ('go tool cover -func=coverage.out ^| find "total"') do set result=%%i
echo Total coverage: %result%
