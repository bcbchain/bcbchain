#!/usr/bin/env bash

uid=$(id -u)
if [[ "${uid:--1}" != "0" ]]; then
  echo "must be root user"
  exit 1
fi 

echo 
echo systemctl stop tmcore
systemctl stop tmcore
echo 

exit 0
