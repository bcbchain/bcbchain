#!/usr/bin/env bash

set -uo pipefail
export PATH=/usr/bin:bin:/usr/sbin:sbin
. common.sh

usage() {
  echo ""
  echo "---------------------------CAUTION---------------------------"
  echo "Please make a install request because bcchain is clean.      "
  echo ""
  exit 1
}

getChainID() {
    echo $(cat /etc/bcchain/genesis 2>/dev/nul)
    return
}

removeOldData() {
  bash clean.sh
}

updateFromOrigin="false"
originChainID=""
if [[ "$*" != "" ]]; then
  originChainID=$1
fi

chainIDDir=$(getChainID)
if [[  -s "/etc/bcchain/genesis" ]] && [[ ! -d ${chainIDDir} ]];then
    echo "this package unprepare for this chain"
    exit 1
fi

if [[ -s prepare-to-update-old-chain.sh ]]; then
  bash prepare-to-update-old-chain.sh ${originChainID}
  ret=$?
  if [[ ${ret} -eq 1 ]]; then
    echo "must prepare to update old chain manually"
    exit 1
  else
    if [[ ${ret} -eq 2 ]]; then
      updateFromOrigin="true"
    fi
  fi 
fi

uid=$(id -u)
if [[ "${uid:-}" != "0" ]]; then
  echo "must be root user"
  exit 1
fi

if $(systemctl -q is-active bcchain.service) ; then
  systemctl stop bcchain.service
fi

uid=$(id -u bcchain 2>/dev/null)
if [[ "${uid:-}" != "0" ]];then
  pid=$(ps -fubcchain 2>/dev/null|grep 'bcchain'|awk '$0 !~/grep/ {print $2}'|sed -e 's/\n/ /')
  if [[ "${pid:-}" != "" ]]; then
      echo "kill old process. ${pid}"
      kill -9 ${pid}
  fi
fi

oldVersion=0.0.0.0
if [[ ${updateFromOrigin} = "false" ]] ; then
  isEmpty=$(isEmptyChain)
  if [[ ${isEmpty} = "true" ]] ; then
    usage
  fi

  if [[ ${emptyBcchain} = "false" ]]; then
    oldVersion=$(/usr/local/bcchain/bin/bcchain version)
  fi
  
  isCorrupted=$(isCorruptedChain)
  if [[ ${isCorrupted} = "true" ]] ; then
    echo ""
    echo Old chain or data is corrupted, do you want to remove all of this chain to reinstall?
    options=("yes" "no")
    select opt in "${options[@]}" ; do
    case ${opt} in
      "yes")
        echo "Yes, remove all of this chain to reinstall"     
        rm -rf /home/bcchain /usr/local/bcchain /etc/bcchain

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
        break
        ;;
      "no")
        echo "No, keep old chain, need to be repaired manually"
        echo ""
        exit 1
        break
        ;;
      *) echo "Invalid choice.";;
      esac
    done
  fi
  
  isCompleted=$(isCompletedChain)
  if [[ ${isCompleted} = "true" ]] ; then
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
else
  oldVersion=1.0.*
fi

chainID=$(getChainID)
doCopyFiles ${chainID}

version=$(head -1 version | tr -d "\r")

echo ""
echo "Congratulation !!! BCCHAIN is successfully updated from ${oldVersion} to version ${version}."
echo ""
