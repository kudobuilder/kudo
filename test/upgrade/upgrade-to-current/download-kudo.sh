#!/usr/bin/env bash

PREV_KUDO_VERSION=$1

echo "Downloading KUDO v${PREV_KUDO_VERSION}"

# Download previous KUDO version for upgrade testing
if [[ "$(uname)" == "Darwin" ]]; then
    curl -L https://github.com/kudobuilder/kudo/releases/download/v${PREV_KUDO_VERSION}/kubectl-kudo_${PREV_KUDO_VERSION}_darwin_x86_64 --output old-kudo
elif [[ "$(expr substr $(uname -s) 1 5)" == "Linux" ]]; then
    curl -L https://github.com/kudobuilder/kudo/releases/download/v${PREV_KUDO_VERSION}/kubectl-kudo_${PREV_KUDO_VERSION}_linux_x86_64 --output old-kudo
fi

chmod +x old-kudo