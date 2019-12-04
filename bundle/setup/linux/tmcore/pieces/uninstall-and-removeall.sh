#!/usr/bin/env bash

echo ""
echo "Do you want to uninstall & remove all of this tendermint node?"
options=("yes" "no")
select opt in "${options[@]}" ; do
case ${opt} in
  "yes")
    echo ""
    echo "Yes, uninstall & remove all of this tendermint node"     
    if $(systemctl -q is-active tmcore.service) ; then
      systemctl stop tmcore.service
    fi 
    rm -fr /etc/tmcore
    rm -fr /home/tmcore
    rm -fr /usr/local/tmcore
    rm -fr /etc/systemd/system/tmcore.service.d
    echo ""
    break
    ;;
  "no")
    echo ""
    echo "No, keep the old tendermint node"
    echo ""
    exit 1
    break
    ;;
  *) echo "Invalid choice.";;
  esac
done 
