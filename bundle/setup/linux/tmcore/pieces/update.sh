#!/usr/bin/env bash

set -uo pipefail
export PATH=/usr/bin:/bin:/usr/sbin:/sbin
. common.sh

usage() {
  echo ""
  echo "---------------------------CAUTION---------------------------"
  echo "Please make a install request because tmcore is clean.       "
  echo ""
  exit 1
}

getChainID() {
  if [[ "${emptyGenesisFile}" = "true" ]] ; then
    echo "can not read genesis.json"
    exit 1
  else
    echo $(cat /etc/tmcore/config/genesis.json | ./jq .chain_id | tr -d "\"")
  fi
}

removeOldData() {
  bash clean.sh
}

uid=$(id -u)
if [[ "${uid:-}" != "0" ]]; then
  echo "must be root user"
  exit 1
fi

chainIDDir=$(getChainID)
if [[ ! -d ${chainIDDir} ]];then
    echo "this package unprepare for this chain"
    exit 1
fi

if $(systemctl -q is-active tmcore.service 2>/dev/null) ; then
  systemctl stop tmcore.service
fi

uid=$(id -u tmcore 2>/dev/null)
if [[ "${uid:-}" != "0" ]];then
  pid=$(ps -futmcore 2>/dev/null|grep 'tendermint'|awk '$0 !~/grep/ {print $2}'|sed -e 's/\n/ /')
  if [[ "${pid:-}" != "" ]]; then
      echo "kill old process. ${pid}"
      kill -9 ${pid}
  fi
fi

isEmpty=$(isEmptyNode)
if [[ "${isEmpty}" = "true" ]] ; then
  usage
fi

oldVersion=0.0.0.0
if [[ "${emptyTendermint}" = "false" ]]; then
  oldVersion=$(/usr/local/tmcore/bin/tendermint version | grep "build version: " | tr -d "build version: ")
fi

isCompletedVersion1=$(isCompletedVersion1Node)
isCorrupted=$(isCorruptedNode)
if [[ "${isCompletedVersion1}" = "false" ]] && [[ "${isCorrupted}" = "true" ]] ; then
  echo ""
  echo Old node or data is corrupted, do you want to remove all of this node to reinstall?
  options=("yes" "no")
  select opt in "${options[@]}" ; do
  case ${opt} in
    "yes")
      echo "Yes, remove all of this node to reinstall"      
      rm -rf /home/tmcore /usr/local/tmcore /etc/tmcore

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
      break
      ;;
    "no")
      echo "No, keep old node and data, need to be repaired manually"
      echo ""
      exit 1
      break
      ;;
    *) echo "Invalid choice.";;
    esac
  done
fi

if [[ "${isCompletedVersion1}" = "true" ]] ; then
  echo ""
  echo Old version node exists, do you want to update?
  options=("yes" "no")
  select opt in "${options[@]}" ; do
  case ${opt} in
    "yes")
      echo "Yes, update the old version node"     
      echo ""
      mv /etc/tmcore/config/genesis-signature.json /etc/tmcore/config/genesis.json.sig
      rm -fr /etc/tmcore/*.json
      break
      ;;
    "no")
      echo "No, keep old version node"
      echo ""
      exit 1
      break
      ;;
    *) echo "Invalid choice.";;
    esac
  done
fi

isCompleted=$(isCompletedNode)
if [[ "${isCompleted}" = "true" ]] ; then
  echo ""
  echo Old data exists, do you want to remove all data to re-sync?
  options=("yes" "no")
  select opt in "${options[@]}" ; do
  case ${opt} in
    "yes")
      echo "Yes, remove old data"
      echo ""
      removeOldData
      break
      ;;
    "no")
      echo "No, keep old data"
      echo ""
      break
      ;;
    *) echo "Invalid choice.";;
    esac
  done
fi

chainID=$(getChainID)
doCopyFiles ${chainID}

version=$(head -1 version | tr -d "\r")

rm -fr /etc/tmcore/genesis/${chainID}/${chainID}-nodes.json >/dev/null 2>/dev/null
rm -fr /etc/tmcore/genesis/${chainID}/${chainID}-watchers.json >/dev/null 2>/dev/null
rm -fr /etc/tmcore/genesis/${chainID}/tendermint-forks.* >/dev/null 2>/dev/null

echo ""
echo "Congratulation !!! TENDERMINT is successfully updated from ${oldVersion} to version ${version}."
echo ""

sed -i 's/recheck = true/recheck = false/' /etc/tmcore/config/config.toml
