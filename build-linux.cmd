echo off

rem build front page
call yarn build
robocopy .\build .\srv\static /MIR
rmdir /s /Q build

rem build server app
mkdir dist
cd srv
set GOOS=linux
set GOARCH=amd64
go build -o ../dist 
cd ..