#!/usr/bin/env bash

set -uo pipefail

export PATH=/usr/local/bcchain/bin:/usr/bin:/bin:/usr/sbin:/sbin

if [[ "${1:-}" == "follow" ]]; then
    officials=$2

    echo ""
    echo "Initializing genesis info..."
    bcchain init --follow ${officials}
    echo ""
    echo ""
fi
