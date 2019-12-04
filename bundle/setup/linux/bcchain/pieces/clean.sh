#!/usr/bin/env bash

removeOldData() {
  j=#
  echo ${j}!/usr/bin/env bash >/etc/bcchain/.clean
	echo /usr/local/bcchain/bin/bcchain unsafe_reset_all >>/etc/bcchain/.clean
	chmod +x /etc/bcchain/.clean
  su - bcchain -s /bin/bash -c "/etc/bcchain/.clean"
  echo "Old data has been removed"
  echo ""
}
removeOldData
