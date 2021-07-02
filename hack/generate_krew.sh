#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script generates a Krew-compatible plugin manifest. It should be run after goreleaser.

VERSION=${VERSION:-$(git describe --tags | sed 's/^v//g')}

# Generate the manifest for a single platform.
function generate_platform {
    ARCH="${2}"
    if [ "${2}" == "amd64" ]; then
        ARCH=x86_64
    elif [ "${2}" == "386" ]; then
        ARCH=i386
    fi

    local sha
    PLATFORM=$(uname)
    if [ "$PLATFORM" == 'Darwin' ]; then
       sha=$(curl -L https://github.com/kudobuilder/kudo/releases/download/v"${VERSION}"/kudo_"${VERSION}"_"${1}"_"${ARCH}".tar.gz | shasum -a 256 - | awk '{print $1}')
    else
        sha=$(curl -L https://github.com/kudobuilder/kudo/releases/download/v"${VERSION}"/kudo_"${VERSION}"_"${1}"_"${ARCH}".tar.gz | sha256sum - | awk '{print $1}')
    fi

    cat <<EOF
  - selector:
      matchLabels:
        os: "${1}"
        arch: "${2}"
    uri: https://github.com/kudobuilder/kudo/releases/download/v${VERSION}/kudo_${VERSION}_${1}_${ARCH}.tar.gz
    sha256: "${sha}"
    bin: "${3}"
    files:
    - from: "*"
      to: "."
EOF
}

rm -f kudo.yaml

# shellcheck disable=SC2129
cat <<EOF >> kudo.yaml
apiVersion: krew.googlecontainertools.github.com/v1alpha2
kind: Plugin
metadata:
  name: kudo
spec:
  version: "v${VERSION}"

  shortDescription: Declaratively build, install, and run operators using KUDO.
  homepage: https://kudo.dev/
  description: |
    The Kubernetes Universal Declarative Operator (KUDO) is a highly productive
    toolkit for writing operators for Kubernetes. Using KUDO, you can deploy
    your applications, give your users the tools they need to operate it, and
    understand how it's behaving in their environments — all without a PhD in
    Kubernetes.

    Example usage:
      Install kafka:
        kubectl kudo install kafka
      List installed operator instances:
        kubectl kudo get instances
    See the documentation for more information: https://kudo.dev/docs/
  caveats: |
    Requires the KUDO controller to be installed:
      kubectl kudo init
  platforms:
EOF

generate_platform linux amd64 ./kubectl-kudo >> kudo.yaml
generate_platform linux 386 ./kubectl-kudo >> kudo.yaml
generate_platform darwin amd64 ./kubectl-kudo >> kudo.yaml
generate_platform darwin arm64 ./kubectl-kudo >> kudo.yaml

### KUDO is not currently built for Windows. Uncomment once it is.
# generate_platform windows amd64 ./kubectl-kudo.exe >> kudo.yaml
# generate_platform windows 386 ./kubectl-kudo.exe >> kudo.yaml

echo "To publish to the krew index, create a pull request to https://github.com/kubernetes-sigs/krew-index/tree/master/plugins to update kudo.yaml with the newly generated kudo.yaml."
