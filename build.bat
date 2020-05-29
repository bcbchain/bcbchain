echo off

cd > cwd.txt
set /p CWD=<cwd.txt
del /q cwd.txt
set GOPATH=%CWD%;%CWD%\..\..\third-party;%CWD%\..\bcsmc-sdk;%CWD%\..\bclib

echo go install ./src/...
go install ./src/...
