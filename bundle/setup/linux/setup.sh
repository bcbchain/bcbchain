#!/usr/bin/env bash

set -o pipefail -o noclobber -o nounset

! getopt --test > /dev/null
if [[ ${PIPESTATUS[0]} -ne 4 ]]; then
    echo '`getopt --test` failed, please install util-linux first'
    exit 1
fi

OPTIONS=t:c:
LONGOPTS=tmcore-data:,chain-data:

# -regarding ! and PIPESTATUS see above
# -temporarily store output to be able to check for errors
# -activate quoting/enhanced mode (e.g. by writing out “--options”)
# -pass arguments only via   -- "$@"   to separate them correctly
! PARSED=$(getopt --options=${OPTIONS} --longoptions=${LONGOPTS} --name "$0" -- "$@")
if [[ ${PIPESTATUS[0]} -ne 0 ]]; then
    # e.g. return value is 1
    #  then getopt has complained about wrong arguments to stdout
    exit 2
fi
# read getopt’s output this way to handle the quoting right:
eval set -- "${PARSED}"

TMCoreData=
ChainData=
# now enjoy the options in order and nicely split until we see --
while true; do
    case "$1" in
        -t|--tmcore-data)
            TMCoreData="$2"
            shift 2
            ;;
        -c|--chain-data)
            ChainData="$2"
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "Wrong parameter: $1"
            exit 3
            ;;
    esac
done

# echo "TMCoreData  = '${TMCoreData}'"
# echo "ChainData   = '${ChainData}'"
if [[ -n ${1:-} ]]; then
    echo "Unknown argument: $1"
    echo ""
    echo "Usage: $0 -t your-tmcore-data-file.tar.gz -c your-chain-data-file.tar.gz"
    echo "       or"
    echo "       $0 --tmcore-data=your-tmcore-data-file.tar.gz --chain-data=your-chain-data-file.tar.gz"
fi

if [[ "${TMCoreData}" != "" ]]; then
    if [[ ! -f "${TMCoreData}" ]]; then
        echo "No such file ${TMCoreData}"
        exit 3
    else
        if [[ "$(tar tf ${TMCoreData}|grep -E "data\/state.db\/CURRENT|data\/blockstore.db\/CURRENT"|wc -l)" != "2" ]]; then
            echo "corrupt tmcore data file, exiting"
            exit 3
        fi
    fi
fi
if [[ "${ChainData}" != "" ]]; then
    if [[ ! -f "${ChainData}" ]]; then
        echo "No such file ${ChainData}"
        exit 3
    else
        if [[ "$(tar tf ${ChainData}|grep -E "\.appstate\.db\/CURRENT|\.appstate\.db\/MANIFEST"|wc -l)" != "2" ]]; then
            echo "corrupt chain data file, exiting"
            exit 3
        fi
    fi
fi

# exit 0

sudo systemctl stop    tmcore 2>/dev/null
sudo systemctl disable tmcore 2>/dev/null

sudo systemctl stop    bcchain 2>/dev/null
sudo systemctl disable bcchain 2>/dev/null

source tmcore_0.0.0.0/pieces/common.sh
source bcchain_0.0.0.0/pieces/common.sh

echo 
echo SETUP bcchain...
if [[ "$(isCorruptedNode)" != "true" ]] && [[ "$(isCompletedNode)" != "true" ]] ; then
    (cd bcchain_0.0.0.0/pieces; bash install.sh)
else
    (cd bcchain_0.0.0.0/pieces; bash update.sh)
fi

if [[ "$?" != "0" ]]; then
    exit 1
fi

echo 
echo SETUP tmcore...
if [[ "$(isCorruptedNode)" != "true" ]] && [[ "$(isCompletedNode)" != "true" ]] ; then
    (cd tmcore_0.0.0.0/pieces; bash install.sh)
    if [[ $? -eq 1 ]]; then
        exit
    fi
    sudo su - tmcore -s /bin/bash -c "/usr/local/tmcore/bin/run.sh init"
else
    (cd tmcore_0.0.0.0/pieces; bash update.sh)
fi

if [[ "$?" != "0" ]]; then
    exit 1
fi

if [[ "${TMCoreData}" != "" ]]; then
    rm -rf /home/tmcore/data
    tar xf ${TMCoreData} -C /home/tmcore
    chown -R tmcore:tmcore /home/tmcore/data
fi

if [[ "${ChainData}" != "" ]]; then
    rm -rf /home/bcchain/.appstate.db
    tar xf ${ChainData} -C /home/bcchain
    chown -R bcchain:bcchain /home/bcchain/.appstate.db
fi

if [[ $? -eq 0 ]]; then
    sudo systemctl enable bcchain 2>/dev/null
	sudo systemctl enable tmcore 2>/dev/null
    sudo systemctl start  bcchain 2>/dev/null
	sudo systemctl start  tmcore 2>/dev/null
fi
