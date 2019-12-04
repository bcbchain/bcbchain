#!/usr/bin/env bash
set -uo pipefail

export PATH=/usr/local/bcchain/bin:/usr/bin:/bin:/usr/sbin:/sbin
clean_old() {
    pids=$(ps -fubcchain|grep bcchain.bin.bcchain|awk '$0 !~/grep/ {print $2}'|sed -e 's/\n/ /')
    if [[ "$pids" != "" ]]; then
        echo "kill old process."
        kill -9 ${pids}
    fi
}

start() {
    echo "run app ->->->->->"
    [[ -d "/home/bcchain/log" ]] || ( mkdir /home/bcchain/log; chmod 775 /home/bcchain/log )
    while true
    do
      bcchain start 2>&1 | (TS=`date "+%F %T"`;sed -e "s/^/[${TS}] chain - /") >>/home/bcchain/log/bcchain.out 2>>/home/bcchain/log/bcchain.out
      sleep 1
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
