#!/usr/bin/env bash

getChainID() {
    cd /etc/tmcore/genesis
    dirs=$(ls -d */ | tr -d "\/")
    for d in ${dirs}; do
      echo ${d}
      return
    done
}

set -uo pipefail

if [[ "${TMHOME:-}" == "" ]] && [[ "$HOME" != "/home/$(whoami)" ]]; then
    export TMHOME=/etc/tmcore
fi
export PATH=/usr/local/tmcore/bin:/usr/bin:/bin:/usr/sbin:/sbin

if [[ "${1:-}" == "genesis" ]]; then
    chainID=$(getChainID)
    genesisPath="/etc/tmcore/genesis"

    echo ""
    echo "Initializing all genesis node..."
    tendermint init --chain_id ${chainID} --genesis_path ${genesisPath}
    rm -fr ${genesisPath}/${chainID}/${chainID}-nodes.json >/dev/null 2>/dev/null
    rm -fr ${genesisPath}/${chainID}/${chainID}-watchers.json >/dev/null 2>/dev/null
    rm -fr ${genesisPath}/${chainID}/tendermint-forks.* >/dev/null 2>/dev/null
    echo ""
    echo ""
    version=$(tendermint version | grep "build version: " | tr -d "build version: ")
    echo "Congratulation !!! TENDERMINT is successfully installed with version ${version}."
    echo ""
    exit 0
fi

if [[ "${1:-}" == "follow" ]]; then
    officials=$2

    echo ""
    echo "Initializing all genesis node..."
    tendermint init --follow ${officials}
    echo ""
    echo ""
    version=$(tendermint version | grep "build version: " | tr -d "build version: ")
    echo "Congratulation !!! TENDERMINT is successfully installed with version ${version}."
    echo ""
    exit 0
fi
