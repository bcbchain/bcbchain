#!/usr/bin/env bash

set -uo pipefail

if [[ "${TMHOME:-}" == "" ]] && [[ "$HOME" != "/home/$(whoami)" ]]; then
    export TMHOME=/etc/tmcore
fi
export PATH=/usr/local/tmcore/bin:/usr/bin:/bin:/usr/sbin:/sbin

clean_old() {
    pid=$(ps -futmcore|grep 'tendermint node'|awk '$0 !~/grep/ {print $2}'|sed -e 's/\n/ /')
    if [[ "${pid:-}" != "" ]]; then
        echo "kill old process."
        kill -9 ${pid}
    fi
}

start() {
    echo "run node ->->->->->"
    [[ -d "/home/tmcore/log" ]] || ( mkdir /home/tmcore/log; chmod 775 /home/tmcore/log )
    while true
    do
     rm -f /tmp/tmcore.monit.*  # remove the record of height,
     tendermint node | (TS=`date "+%F %T"`;sed -e "s/^/[${TS}] tmcore - /") >>/home/tmcore/log/tmcore.out 2>>/home/tmcore/log/tmcore.out
     sleep 1
    done
    date >> /home/tmcore/log/tmcore.out
    exit 1
}

getChainID() {
    cd /etc/tmcore/genesis
    dirs=$(ls -d */ | tr -d "\/")
    for d in ${dirs}; do
      echo ${d}
      return
    done
}

if [[ "${1:-}" == "" ]] || [[ "${1:-}" == "start" ]] || [[ "${1:-}" == "restart" ]]; then
    clean_old
    start
    exit 0
fi

if [[ "${1:-}" == "stop" ]]; then
    clean_old
    exit 0
fi
