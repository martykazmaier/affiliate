@echo off
rem EleBBS is a 32-bit process. The door MUST be 32-bit so the
rem inherited WinSock handle in DOOR32.SYS is valid in this process.
set GOOS=windows
set GOARCH=386
go build -ldflags="-s -w" -o affiliate.exe .
if errorlevel 1 exit /b 1
echo Built affiliate.exe (windows/386 for EleBBS sockets)
