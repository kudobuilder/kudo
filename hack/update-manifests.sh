#!/usr/bin/env bash

if ! command -v yq &> /dev/null
then
    echo "required yq command NOT found"
    exit 1
fi

if ! command -v awk &> /dev/null
then
    echo "required awk command NOT found"
    exit 1
fi

# hack/manifest-gen is the build folder
mkdir -p hack/manifest-gen
mkdir -p config/manifests

#  gen manifests to hack/manifest-gen/manifests.yaml
go run cmd/kubectl-kudo/main.go init  --dry-run --version dev --unsafe-self-signed-webhook-ca -o yaml > hack/manifest-gen/manifests.yaml

cd hack/manifest-gen/ || exit

# separate the manifests
awk 'BEGIN{file = 0; filename = "output_" file ".txt"}
    /---$/ {getline; file ++; filename = "output_" file ".txt"}
    {print $0 > filename}' manifests.yaml
cd - || exit

#  loop through all the files
for f in hack/manifest-gen/*.txt; 
do 
    KIND=$(yq r "$f" kind)

case "$KIND" in 
    "CustomResourceDefinition") 
        NAME=$(yq r "$f" spec.names.kind)
        echo "skip '$NAME' crd"
        continue;;

    "StatefulSet") 
        echo "skipping statefulset"
        continue;;
    "Service")
        echo "skipping service"
        continue;;
    "Secret")
        echo "skipping secret"
        continue;;
    "Namespace")
        KIND=ns;;    
    "ServiceAccount")
        KIND=sa;;
    *)
        KIND="";;        
esac

NAME=$(yq r "$f" metadata.name)

if [ -n "$KIND" ]
then
    NAME="$NAME-$KIND"
fi 
NAME="$NAME.yaml"

echo "Working with  $NAME"

cp "$f" "config/manifests/$NAME"
done

# update webhook (add config.url and remove config.caBundle and config.service)
yq w -i config/manifests/kudo-manager-instance-admission-webhook-config.yaml webhooks[0].clientConfig.url https://replace-url.com
yq d -i config/manifests/kudo-manager-instance-admission-webhook-config.yaml webhooks[0].clientConfig.caBundle
yq d -i config/manifests/kudo-manager-instance-admission-webhook-config.yaml webhooks[0].clientConfig.service

rm -rf hack/manifest-gen
echo "Finished"