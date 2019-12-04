#!/usr/bin/env bash

set -uo pipefail
export PATH=/usr/bin:/bin:/usr/sbin:/sbin
. common.sh
. init.sh

usage1() {
  echo ""
  echo "---------------------------CAUTION---------------------------"
  echo "Please make a update request because tmcore exists, but node "
  echo "or data is corrupted."
  echo ""
  exit 1
}

usage2() {
  echo ""
  echo "---------------------------CAUTION---------------------------"
  echo "Please make a update request because tmcore data exists, not "
  echo "a clean install."
  echo ""
  exit 1
}

uid=$(id -u)
if [[ "${uid:-}" != "0" ]]; then
  echo "must be root user"
  exit 1
fi

if $(systemctl -q is-active tmcore.service 2>/dev/null) ; then
  systemctl stop tmcore
  usage2
fi

uid=$(id -u tmcore 2>/dev/null)
if [[ "${uid:-}" != "" ]]; then
  pid=$(ps -futmcore 2>/dev/null|grep 'tendermint'|awk '$0 !~/grep/ {print $2}'|sed -e 's/\n/ /')
  if [[ "${pid:-}" != "" ]]; then
      echo "kill old process. ${pid}"
      kill -9 ${pid}
  fi
fi

isCorrupted=$(isCorruptedNode)
if [[ ${isCorrupted} = "true" ]] ; then
  usage1
fi

isCompleted=$(isCompletedNode)
if [[ ${isCompleted} = "true" ]] ; then
  usage2
fi

echo ""
echo "Select HOW to install tmcore service"
choices=("")
choices[0]="GENESIS VALIDATOR"
choices[1]="FOLLOWER"
select nodeType in "${choices[@]}"; do
    case ${nodeType} in
    "GENESIS VALIDATOR")
        echo "You selected GENESIS VALIDATOR node"
        echo ""
        installGenesisValidator
        ;;
    "FOLLOWER")
        echo "You selected FOLLOWER"
        echo ""
        installFollower
        ;;
    *) echo "Invalid choice.";;
    esac
done

exit 0
