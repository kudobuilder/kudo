#!/bin/bash

# This script generates a Krew-compatible plugin manifest. It should be run after goreleaser.

VERSION=$(git describe --tags |sed 's/^v//g')

# Generate the manifest for a single platform.
function generate_platform {
    ARCH="${2}"
    if [ "${2}" == "amd64" ]; then
        ARCH=x86_64
    elif [ "${2}" == "386" ]; then
        ARCH=i386
    fi

    cat <<EOF
  - selector:
      matchLabels:
        os: "${1}"
        arch: "${2}"
    uri: https://github.com/kudobuilder/kudo/releases/download/v${VERSION}/kudo_${VERSION}_${1}_${ARCH}.tar.gz
    sha256: "$(sha256sum dist/kudo_${VERSION}_${1}_${ARCH}.tar.gz |awk '{print $1}')"
    bin: "./kubectl-kudo"
    files:
    - from: "*"
      to: "."
EOF
}

rm -f kudo.yaml

cat <<EOF >> kudo.yaml
apiVersion: krew.googlecontainertools.github.com/v1alpha2
kind: Plugin
metadata:
  name: kudo
spec:
  version: "${VERSION}"

  shortDescription: KUDO CLI
  homepage: https://kudo.dev/
  description: |
    The Kubernetes Universal Declarative Operator (KUDO) is a highly productive toolkit for writing operators for Kubernetes.
    Using KUDO, you can deploy your applications, give your users the tools they need to operate it, and understand how it's
    behaving in their environments — all without a PhD in Kubernetes. 

  platforms:
EOF

generate_platform linux amd64 >> kudo.yaml
generate_platform linux 386 >> kudo.yaml
generate_platform darwin amd64 >> kudo.yaml
generate_platform darwin 386 >> kudo.yaml
generate_platform windows amd64 >> kudo.yaml
generate_platform windows 386 >> kudo.yaml

echo "To publish to the krew index, create a pull request to https://github.com/kubernetes-sigs/krew-index/tree/master/plugins to update kudo.yaml with the newly generated kudo.yaml."
