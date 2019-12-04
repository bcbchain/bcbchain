#!/bin/bash

sudo systemctl stop    tmcore 2>/dev/null
sudo systemctl disable tmcore 2>/dev/null

cd pieces
bash update.sh
if [[ $? -ne 0 ]]; then
	exit
fi

sudo systemctl enable tmcore 2>/dev/null
sudo systemctl start  tmcore 2>/dev/null
