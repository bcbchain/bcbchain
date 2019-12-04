#!/usr/bin/env bash

echo ""
echo "Do you want to uninstall & remove all of this chain?"
options=("yes" "no")
select opt in "${options[@]}" ; do
case ${opt} in
  "yes")
    echo ""
    echo "Yes, uninstall & remove all of this chain"     
    if $(systemctl -q is-active bcchain.service) ; then
      systemctl stop bcchain.service
    fi 
    rm -fr /etc/bcchain
    rm -fr /home/bcchain
    rm -fr /usr/local/bcchain
    echo ""
    break
    ;;
  "no")
    echo ""
    echo "No, keep the old chain"
    echo ""
    exit 1
    break
    ;;
  *) echo "Invalid choice.";;
  esac
done 
