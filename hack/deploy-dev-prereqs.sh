#!/usr/bin/env bash

if ! command -v kubectl &> /dev/null
then
    echo "required kubectl command NOT found"
    exit 1
fi

# cache folder 
MANCACHE=hack/manifest-gen

if [ ! -d "$MANCACHE" ]
then
    echo "manifests do not exist."
    echo "you must run 'make generate-manifests' or './hack/update-manifests.sh'"
    exit 1
fi


kubectl apply -f "$MANCACHE"/
kubectl apply -f config/crds

echo "Finished"