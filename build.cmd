echo off

rem build front page
call yarn build
robocopy .\build .\srv\static /MIR
rmdir /s /Q build

rem build server app
mkdir dist
cd srv
go build -o ../dist
cd ..