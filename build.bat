echo off

cd > cwd.txt
set /p CWD=<cwd.txt
del /q cwd.txt

SET ThirdpartyDir=.\..\thirdparty-master

if  exist %ThirdpartyDir% (
    set GOPATH=%CWD%;%CWD%\..\thirdparty-master

    echo go install blockchain
    go install blockchain/cmd/bcchain

    echo go install tendermint
    go install github.com/tendermint/tendermint/cmd/tendermint
) else (
    echo "please download thirdparty-master.zip from https://github.com/bcbchain/thirdparty/archive/master.zip"
    echo "if the current directory is in the D:/bcb/bcbchain-master,then the thirdparty code must be in the D:/bcb/"

    pause
)


