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

#emptySdk=$(isEmptyFileOrDir "/home/bcchain/.build/sdk")
#emptyThird=$(isEmptyFileOrDir "/home/bcchain/.build/thirdparty")
emptyBuild=$(isEmptyFileOrDir "/home/bcchain/.build")
emptyClean=$(isEmptyFileOrDir "/home/bcchain/clean.sh")
emptyData=$(isEmptyFileOrDir "/home/bcchain/.appstate.db")
emptyBcchain=$(isEmptyFileOrDir "/usr/local/bcchain/bin/bcchain")

isEmptyChain() {
  if [[ ${emptyBuild} == "true" ]] && [[ ${emptyClean} == "true" ]] && [[ ${emptyData} == "true" ]] && [[ ${emptyBcchain} == "true" ]]; then
    echo "true"
    return
  else
    echo "false"
    return
  fi
}

isCompletedChain() {
  if [[ ${emptyBuild} == "false" ]] && [[ ${emptyClean} == "false" ]] && [[ ${emptyData} == "false" ]] && [[ ${emptyBcchain} == "false" ]]; then
    echo "true"
    return
  else
    echo "false"
    return
  fi
}

isCorruptedChain() {
  t=$(isEmptyChain)
  if [[ ${t} = "true" ]]; then
    echo "false"
    return
  fi
  t=$(isCompletedChain)
  if [[ ${t} = "true" ]]; then
    echo "false"
    return
  fi
  echo "true"
  return
}

getChainInfoFromNode() {
  officials_=$1
  firstNode=$(echo "${officials_}" | cut -d, -f1)
  genesis=$(curl -L http://"${firstNode}"/genesis 2>/dev/null)
  chainID=($(echo "${genesis}" | ./jq '.result.genesis.chain_id' 2>/dev/null | tr -d "\""))
  chainVersion=($(echo "${genesis}" | ./jq '.result.genesis.chain_version' 2>/dev/null | tr -d "\""))
  if [ -z "${chainID:-}" ]; then
     genesis=$(curl https://"${firstNode}"/genesis 2>/dev/null)
     chainID=($(echo "${genesis}" | ./jq '.result.genesis.chain_id' 2>/dev/null | tr -d "\""))
     chainVersion=($(echo "${genesis}" | ./jq '.result.genesis.chain_version' 2>/dev/null | tr -d "\""))
  fi
  eval $2="'${chainID:-}'"
  eval $3="'${chainVersion:-}'"
}

doCopyFiles() {
  chainID=$1

  echo "Start copying files ..."
	
  mkdir -p /etc/bcchain /home/bcchain/{log,.appstate.db,.build/smcrunsvc_v1.0_3dcontract/bin} /usr/local/bcchain/bin
  echo "${chainID}" > /etc/bcchain/genesis
  
  getent group bcchain  >/dev/null 2>&1 || groupadd -r bcchain
  getent passwd bcchain  >/dev/null 2>&1 || useradd -r -g bcchain \
      -d /home/bcchain -s /sbin/nologin -c "BlockChain application System User" bcchain
  usermod -d /home/bcchain -g bcchain bcchain 2>/dev/null

  systemctl stop docker
  rm -f /run/docker.* 2>/dev/null
  systemctl start docker
  usermod -G $(ls -g /run/docker.sock|awk '{print $3}') bcchain 2>/dev/null
    
  cd ${chainID}/.config
  [[ -f abci-forks.json ]] && cp abci-forks.json* /usr/local/bcchain/bin
  cp bcchain.yaml /etc/bcchain
  touch /etc/bcchain/.upgrade
  if [[ "${2:-}" == "genesis" || "${2:-}" == "" ]]; then
    cp *.tar.gz /etc/bcchain
  fi
  cd ../..

  cp smcrunsvc /home/bcchain/.build/smcrunsvc_v1.0_3dcontract/bin
  cp bcchain runApp.sh init.sh /usr/local/bcchain/bin
  cp start.sh stop.sh clean.sh /home/bcchain 
  cp bcchain.service /usr/lib/systemd/system
  
  chown -R bcchain:bcchain /home/bcchain /etc/bcchain /usr/local/bcchain/bin

  chmod 644 /etc/bcchain/*
  chmod 775 /etc/bcchain
  chmod 775 /home/bcchain
  chmod 775 /home/bcchain/log
  chmod 644 /usr/lib/systemd/system/bcchain.service
  chmod 755 /usr/local/bcchain /usr/local/bcchain/bin /usr/local/bcchain/bin/*

#  bash package.sh
  diff version_sdk /home/bcchain/.build/sdk/version >/dev/null 2>/dev/null
  if [[ "$?" != "0" ]]; then
    oldVer=$(cat /home/bcchain/.build/sdk/version 2>/dev/null | tr -d "\r")
    newVer=$(cat version_sdk|tr -d "\r")
    if [[ -z ${oldVer} ]]; then
      echo install sdk with version ${newVer}
    else
      echo update sdk from version ${oldVer} to ${newVer}
    fi
    rm -fr /home/bcchain/.build/sdk 2>/dev/null
    tar xvf sdk*.tar.gz -C /home/bcchain/.build >/dev/null
  fi

  diff version_thirdparty /home/bcchain/.build/thirdparty/version >/dev/null 2>/dev/null
  if [[ "$?" != "0" ]]; then
    oldVer=$(cat /home/bcchain/.build/thirdparty/version 2>/dev/null | tr -d "\r")
    newVer=$(cat version_thirdparty|tr -d "\r")
    if [[ -z ${oldVer} ]]; then
      echo install thirdparty packages with version ${newVer}
    else
      echo update thirdparty packages from version ${oldVer} to ${newVer}
    fi
    rm -fr /home/bcchain/.build/thirdparty 2>/dev/null
    tar xvf third_party*.tar.gz -C /home/bcchain/.build >/dev/null
  fi
  
  echo "End of copy files."
  
  systemctl daemon-reload
}

installGenesisValidator() {
  echo "Select which CHAINID to install"
  dirs=$(ls -d */ | tr -d "\/" | sed '/\[/d')
  choices1=(${dirs})
  select chainID in "${choices1[@]}"; do
    [[ -n ${chainID} ]] || { echo "Invalid choice." >&2; continue; }
    echo "You selected CHAINID=${chainID}"
    echo ""
    doCopyFiles ${chainID} "genesis"
    break
  done
  
  echo ""
  version=$(./bcchain version | tr -d "\r")
  echo "Congratulation !!! BCCHAIN is successfully installed with version ${version}."
  echo ""
  exit 0
}

installFollower() {
  echo "Please input the which node or FOLLOWER's name[:port] you want to follow"
  read -p "node or FOLLOWER to follow: " official
  echo ""
  echo "You selected \"${official}\" to follow"
  echo ""
  chainID=
  chainVersion=
  getChainInfoFromNode ${official} chainID chainVersion
  if [ "${chainID}" == "" ]; then
    echo "Cannot get chainID from ${official}"
    echo ""
    exit 1
  fi
  echo "CHAINID=${chainID}"
  
  doCopyFiles ${chainID} ${chainVersion}
  su - bcchain -s /bin/bash -c "/usr/local/bcchain/bin/init.sh follow ${official}"

  echo ""
  version=$(./bcchain version | tr -d "\r")
  echo "Congratulation !!! BCCHAIN is successfully installed with version ${version}."
  echo ""
  exit 0
}
