#!/usr/bin/env bash

removeOldData() {
  echo "TMHOME=/etc/tmcore /usr/local/tmcore/bin/tendermint unsafe_reset_all" >/etc/tmcore/.clean
  chmod +x /etc/tmcore/.clean
  su - tmcore -s /bin/bash -c "/etc/tmcore/.clean"
  echo "Old data has been removed"
  echo "" 
}
removeOldData
