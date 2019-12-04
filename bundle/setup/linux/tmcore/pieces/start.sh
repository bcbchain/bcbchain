#!/usr/bin/env bash

uid=$(id -u)
if [[ "${uid:--1}" != "0" ]]; then
  echo "must be root user"
  exit 1
fi 

echo 
echo systemctl start tmcore
systemctl start tmcore
echo 

exit 0
