#!/usr/bin/env bash

set -uo pipefail

export PATH=/usr/bin:/bin:/usr/sbin:/sbin

[[ ! -d /home/tmcore/log ]] && mkdir -p /home/tmcore/log && chown tmcore:tmcore /home/tmcore/log >/dev/null

if [ "$(systemctl is-enabled tmcore.service 2>/dev/null)" != "enabled" ]; then
  echo noService >>/home/tmcore/log/rutaller.log
  exit 0
fi

protocol="http://"
[[ -f /etc/tmcore/config/STAR.bcbchain.io.crt ]] && protocol="https://"

myPort=$(grep -m 1 "^laddr =" /etc/tmcore/config/config.toml | awk -F: '{print $3}' | sed -e 's/"//')

myStatusTempFile=$(mktemp /tmp/tmcore-myStatus-XXXXXXXX)
(curl --max-time 3 --connect-timeout 2 --silent -k "${protocol}localhost:${myPort}/status" 2>&1) >"${myStatusTempFile}"
imSyncing=$(awk '/syncing/{print $2}' "${myStatusTempFile}")

myHeight=$(awk '/latest_block_height/{gsub(",","");print $2}' "${myStatusTempFile}")
myHeight=${myHeight:--1}
rm -f "${myStatusTempFile}"

now=$(date +%s)

stopIt() {
  # 能取点儿调试信息吗
  # curl -s localhost:2020/debug/pprof/heap -o /home/tmcore/log/heap."${now}"
  # curl -s localhost:2020/debug/pprof/allocs -o /home/tmcore/log/allocs."${now}"
  # curl -s localhost:2020/debug/pprof/goroutine\?debug=2 -o /home/tmcore/log/goroutine."${now}"
  # curl -s localhost:2020/debug/pprof/profile -o /home/tmcore/log/profile."${now}"
  # curl -s localhost:2020/debug/pprof/trace -o /home/tmcore/log/trace."${now}"
  # 取证结束
  pid=$(pgrep -u "$(id -u tmcore)" -f "tendermint node" | xargs)
  if [[ "${pid:-}" != "" ]]; then
    echo "$(date "+%F %T") kill tendermint" >>/home/tmcore/log/rutaller.log
    kill -HUP "${pid}"
  fi
  sleep 5 # 给你5秒自行了断
  pid2=$(pgrep -u "$(id -u tmcore)" -f "tendermint node" | xargs)
  if [[ "${pid:-}" == "${pid2:-0}" ]]; then
    echo "$(date "+%F %T") how old are you, 怎么老是你！"
    echo "$(date "+%F %T") kill same tendermint " >>/home/tmcore/log/rutaller.log
    kill -9 "${pid2}" # 你再不死我也没辙了
  fi
  echo >&2 "$(date "+%F %T") restarted"
}

iGrow() {
  echo "${myHeight}" >/tmp/tmcore.monit."${now}"
  chown tmcore:tmcore /tmp/tmcore.monit."${now}"
  for f in /tmp/tmcore.monit.*; do
    t=$(echo "${f}" | cut -d'.' -f 3)
    lifeOfSecond=$((now - t))
    if [[ ${lifeOfSecond} -gt 1200 ]]; then
      rm -f "${f}"
      continue
    elif [[ ${lifeOfSecond:-0} -gt 600 ]]; then
      historyHeight=$(cat "${f}")
      grow=$((myHeight - historyHeight))
      if [[ ${grow} -eq 0 ]]; then
        rm -f /tmp/tmcore.monit.*
        echo "$(date "+%F %T") myHeight is not grow, kill" >>/home/tmcore/rutaller.log
        stopIt
        return
      fi
    fi
  done
  # echo 0
}

if [[ ${myHeight} -eq -1 ]]; then
  initing=$(/usr/local/tmcore/bin/jq .last_height /etc/tmcore/config/priv_validator.json)
  if [[ "${initing}" == "0" || "${initing}" == "null" ]]; then
    exit 0
  else
    echo "$(date "+%f %t") not get myHeight, kill" >>/home/tmcore/rutaller.log
    stopIt
  fi
fi

if ! ${imSyncing:-false}; then
  PEERS=$(curl --max-time 3 --connect-timeout 2 --silent -k ${protocol}localhost:"${myPort}"/net_info | awk -F\" '/listen_addr/{print $4}')
  peerHeight=0

  for PEER in ${PEERS}; do
    P_IP="$(echo "${PEER}" | cut -d':' -f1)"
    P_PORT="$(echo "${PEER}" | cut -d':' -f2)"
    P_PORT=$((P_PORT + 1))
    peerStatusTempFile=$(mktemp /tmp/tmcore-peerStatus-XXXXXXXX)
    if [[ $P_IP == *"-p2p"* ]]; then
      P_ADDR="https://${P_IP/-p2p/}"
    else
      P_ADDR="http://${P_IP}:${P_PORT}"
    fi
    TMPJSON=$(curl --max-time 3 --connect-timeout 2 --silent -k "${P_ADDR}"/status 2>&1)
    if [ -z "$TMPJSON" ]; then
      (curl --max-time 3 --connect-timeout 2 --silent -k "${P_ADDR}"/status 2>&1) >"${peerStatusTempFile}"
    else
      (curl --max-time 3 --connect-timeout 2 --silent -k "${P_ADDR}"/status 2>&1) >"${peerStatusTempFile}"
    fi
    peerSyncing=$(awk '/syncing/{print $2}' "${peerStatusTempFile}")
    if ${peerSyncing:-false}; then
      rm -f "${peerStatusTempFile}"
      continue
    fi
    peerHeight=$(awk '/latest_block_height/{gsub(",","");print $2}' "${peerStatusTempFile}")
    peerHeight=${peerHeight:--1}
    rm -f "${peerStatusTempFile}"
    if [[ ${peerHeight} -ne -1 ]]; then
      break
    fi
  done

  if [[ ${peerHeight} -ne -1 ]]; then
    subVal=$((${peerHeight:-0} - myHeight))
    # echo $subVal
    if [[ ${subVal} -gt 10 ]]; then
      echo "$(date "+%F %T") Consensus is 10 floors lower than neighbours,kill" >>/home/tmcore/log/rutaller.log
      stopIt
    fi
  fi
fi

iGrow
# iii=$(iGrow)
# if [ $iii -ne 0 ]; then
#    exit -1
# fi

exit 0
