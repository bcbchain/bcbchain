#!/usr/bin/env bash

newName=""
if [ -d /etc/bcchain ] || [ -d /usr/local/bcchain ] || [ -d /home/bcchain ]; then
  newName="bcchain"
fi
if [ "${newName}" != "" ]; then
  exit 0
fi

originChainID=$1
originName=""
if [ -d /etc/bcbchain ] && [ -d /usr/local/bcbchain ] && [ -d /home/bcbchain ]; then
  originChainID="bcb"
  originName="bcbchain"
fi
if [ -d /etc/bcbtestchain ] && [ -d /usr/local/bcbtestchain ] && [ -d /home/bcbtestchain ]; then
  originChainID="bcbtest"
  originName="bcbtestchain"
fi
if [ -d /etc/gichain ] && [ -d /usr/local/gichain ] && [ -d /home/gichain ]; then
  originName="gichain"
fi

if [ "${originChainID}" = "" ] ; then
  echo "not found chain id"
  exit 1
fi
if [ "${originName}" = "" ] ; then
  echo "not found chain directory completely"
  exit 1
fi

if $(systemctl -q is-active ${originName}.service) ; then
  systemctl stop ${originName}.service
fi
mv /etc/${originName} /etc/bcchain
mv /usr/local/${originName} /usr/local/bcchain
mv /home/${originName} /home/bcchain

echo ${originChainID} > /etc/bcchain/genesis
if [ -s /etc/bcchain/${originName}.yaml ]; then
  mv /etc/bcchain/${originName}.yaml /etc/bcchain/bcchain.yaml
fi
if [ -s /usr/local/bcchain/bin/${originName} ]; then
  mv /usr/local/bcchain/bin/${originName} /usr/local/bcchain/bin/bcchain
fi
if [ -s /usr/local/bcchain/bin/abci-forks-signature.json ]; then
  mv /usr/local/bcchain/bin/abci-forks-signature.json /usr/local/bcchain/bin/abci-forks.json.sig
fi

exit 2
