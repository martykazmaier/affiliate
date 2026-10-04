@echo off
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o affiliate-linux-amd64 .
if errorlevel 1 exit /b 1
set GOARCH=386
go build -ldflags="-s -w" -o affiliate-linux-386 .
if errorlevel 1 exit /b 1
set GOARCH=arm64
go build -ldflags="-s -w" -o affiliate-linux-arm64 .
if errorlevel 1 exit /b 1
echo Built affiliate-linux-amd64 affiliate-linux-386 affiliate-linux-arm64
set GOOS=
set GOARCH=
