#!/usr/bin/env bash

COLUMNS=12

isEmptyFileOrDir() {
  dir=$1
  if [[ -z "$(ls -A ${dir} 2>/dev/null)" ]]; then
    echo "true"
  else
    echo "false"
  fi
}

emptyClean=$(isEmptyFileOrDir "/home/tmcore/clean.sh")
emptyGenesisFile=$(isEmptyFileOrDir "/etc/tmcore/config/genesis.json")
emptyTendermint=$(isEmptyFileOrDir "/usr/local/tmcore/bin/tendermint")
emptyNodeKey=$(isEmptyFileOrDir "/etc/tmcore/config/node_key.json")
emptyPrivKey=$(isEmptyFileOrDir "/etc/tmcore/config/priv_validator.json")
emptyGenesisDir=$(isEmptyFileOrDir "/etc/tmcore/genesis")

isEmptyNode() {
  if [[ ${emptyClean} == "true" ]] && [[ ${emptyTendermint} == "true" ]] && [[ ${emptyNodeKey} == "true" ]] && [[ ${emptyPrivKey} == "true" ]]; then
    echo "true"
    return
  else
    echo "false"
    return
  fi
}

isCompletedVersion1Node() {
  if [[ ${emptyClean} == "true" ]] && [[ ${emptyGenesisDir} == "true" ]] && [[ ${emptyGenesisFile} == "false" ]] && [[ ${emptyTendermint} == "false" ]] && [[ ${emptyNodeKey} == "false" ]] && [[ ${emptyPrivKey} == "false" ]]; then
    echo "true"
    return
  else
    echo "false"
    return
  fi
}

isCompletedNode() {
  if [[ ${emptyClean} == "false" ]] && [[ ${emptyGenesisFile} == "false" ]] && [[ ${emptyTendermint} == "false" ]] && [[ ${emptyNodeKey} == "false" ]] && [[ ${emptyPrivKey} == "false" ]]; then
    echo "true"
    return
  else
    echo "false"
    return
  fi
}

isCorruptedNode() {
  t=$(isEmptyNode)
  if [[ ${t} = "true" ]]; then
    echo "false"
    return
  fi
  t=$(isCompletedNode)
  if [[ ${t} = "true" ]]; then
    echo "false"
    return
  fi
  echo "true"
  return
}

getChainIDFromNode() {
  officials_=$1
  firstNode=$(echo ${officials_} | cut -d, -f1) 
  genesis=$(curl -L http://${firstNode}/genesis 2>/dev/null)
  chainID=($(echo ${genesis} | ./jq '.result.genesis.chain_id' 2>/dev/null | tr -d "\""))
  if [ -z ${chainID:-} ]; then
     genesis=$(curl https://${firstNode}/genesis 2>/dev/null)
     chainID=($(echo ${genesis} | ./jq '.result.genesis.chain_id' 2>/dev/null | tr -d "\""))
  fi
  echo ${chainID:-}
}

doCopyFiles() {
  chainID_=${1:-}
  version_=${2:-}
  
  echo "Start copying files ..."
  if [[ "${version_}" == "" ]]; then
  
    # for follower or update
    mkdir -p mkdir -p /etc/tmcore/genesis /home/tmcore/data /home/tmcore/log /usr/local/tmcore/bin
    mkdir -p /etc/systemd/system/tmcore.service.d
      
    getent group tmcore  >/dev/null 2>&1 || groupadd -r tmcore
    getent passwd tmcore >/dev/null 2>&1 || useradd -r -g tmcore \
      -d /etc/tmcore -s /sbin/nologin -c "BlockChain.net tendermint core System User" tmcore
    
    cp jq tendermint p2p_ping init.sh run.sh rutaller.bash /usr/local/tmcore/bin
    cp start.sh stop.sh clean.sh /home/tmcore
    cp tmcore.service /usr/lib/systemd/system
    cp override.conf /etc/systemd/system/tmcore.service.d

    cd ${chainID_}
    [[ -f tendermint-forks.json ]] && cp tendermint-forks.json* /usr/local/tmcore/bin
    cd ..

  else
    
    # for genesis validator
    mkdir -p /etc/tmcore/genesis/${chainID_} /home/tmcore/data /home/tmcore/log /usr/local/tmcore/bin
    mkdir -p /etc/systemd/system/tmcore.service.d
      
    getent group tmcore  >/dev/null 2>&1 || groupadd -r tmcore
    getent passwd tmcore  >/dev/null 2>&1 || useradd -r -g tmcore \
      -d /etc/tmcore -s /sbin/nologin -c "BlockChain.net tendermint core System User" tmcore
    
    cp jq tendermint p2p_ping init.sh run.sh rutaller.bash /usr/local/tmcore/bin
    cp start.sh stop.sh clean.sh /home/tmcore
    cp tmcore.service /usr/lib/systemd/system
    cp override.conf /etc/systemd/system/tmcore.service.d

    cd ${chainID_}
    [[ -f tendermint-forks.json ]] && cp tendermint-forks.json* /usr/local/tmcore/bin
    cd ..

    pushd ${chainID_}/${version_} >/dev/null
    tar cvf - * 2>/dev/null |(cd /etc/tmcore/genesis/${chainID_};tar xvf - >/dev/null )
    popd >/dev/null
    
    echo ${version_} > /etc/tmcore/genesis/${chainID_}/genesis.version
    chmod 777 /etc/tmcore/genesis/${chainID_}/genesis.version
  fi

  export TMHOME=/etc/tmcore
  touch /var/spool/cron/root
  sed -i '/rutaller.bash/d' /var/spool/cron/root
  echo "* * * * * /usr/local/tmcore/bin/rutaller.bash >> /home/tmcore/log/rutaller.log 2>&1" >> /var/spool/cron/root
  chown -R tmcore:tmcore /etc/tmcore
  chown -R tmcore:tmcore /home/tmcore/data
  chown tmcore:tmcore /home/tmcore/log
  chown tmcore:tmcore /home/tmcore
  chown tmcore:tmcore /usr/local/tmcore/bin
  
  chmod 600 /var/spool/cron/root
  chmod 755 /home/tmcore/data
  chmod 775 /home/tmcore
  chmod 775 /home/tmcore/log
  chmod 644 /usr/lib/systemd/system/tmcore.service
  chmod 755 /etc/tmcore /usr/local/tmcore /usr/local/tmcore/bin /usr/local/tmcore/bin/*
  
  echo "End of copy files."
  
  systemctl daemon-reload
}

installGenesisValidator() {
  echo "Select which CHAINID to install"
  dirs=$(ls -d */ | tr -d "\/" | sed '/\[/d')
  dirs2=""
  for i in ${dirs}
  do
    if [[ -d ${i}/v1 ]] || [[ -d ${i}/v2 ]] ; then
      dirs2="$dirs2 $i"
    fi
  done
  choices1=(${dirs2})
  select chainID in "${choices1[@]}"; do
    [[ -n ${chainID} ]] || { echo "Invalid choice." >&2; continue; }
    echo "You selected CHAINID=${chainID}"
    echo ""
    break
  done
  
  dirs=$(pushd ${chainID} >/dev/null ; ls -d */ | tr -d "\/" ; popd >/dev/null)
  choices1=(${dirs})
  len=${#choices1[*]}
  if [[ ${len} == 1 ]]; then
    genesisVersion=${dirs}
  else
    echo "Select which GENESIS-VERSION to install"
    select genesisVersion in "${choices1[@]}"; do
      [[ -n ${genesisVersion} ]] || { echo "Invalid choice." >&2; continue; }
      echo "You selected GENESIS-VERSION=${genesisVersion}"
      echo ""
      break
    done
  fi
  
  doCopyFiles ${chainID} ${genesisVersion}
  su - tmcore -s /bin/bash -c "/usr/local/tmcore/bin/init.sh genesis"
  exit $?
}

installFollower() {
  echo "Please input the which node or FOLLOWER's name[:port] you want to follow"
  echo "Multi nodes can be separated by comma \",\""
  echo "for example \"earth.bcbchain.io,mar.bcbchain.io:46657\" or \"venus.bcbchain.io\""
  read -p "nodes or FOLLOWERs to follow: " officials
  echo ""
  echo "You selected \"${officials}\" to follow"
  echo ""
  chainID=$(getChainIDFromNode ${officials})
  if [ "${chainID}" == "" ]; then
    echo "Cannot get chainID from ${officials}"
    echo ""
    exit 1
  fi
  echo "CHAINID=${chainID}"
  
  doCopyFiles ${chainID}
  su - tmcore -s /bin/bash -c "/usr/local/tmcore/bin/init.sh follow ${officials}"
  exit $?
}
