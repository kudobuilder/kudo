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

# cache folder 
MANCACHE=hack/manifest-gen
TEMP="$MANCACHE/temp"

# hack/manifest-gen is the build folder
mkdir -p "$MANCACHE"
mkdir -p "$TEMP"

#  gen manifests to hack/manifest-gen/manifests.yaml
go run cmd/kubectl-kudo/main.go init  --dry-run --version dev --unsafe-self-signed-webhook-ca -o yaml > "$TEMP/manifests.yaml"

cd "$TEMP" || exit

# separate the manifests
awk 'BEGIN{file = 0; filename = "output_" file ".txt"}
    /---$/ {getline; file ++; filename = "output_" file ".txt"}
    {print $0 > filename}' manifests.yaml
cd - || exit

#  loop through all the files
for f in "$TEMP"/*.txt; 
do 
    KIND=$(yq '.kind' "$f")

case "$KIND" in 
    "CustomResourceDefinition") 
        NAME=$(yq '.spec.names.kind' "$f")
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

NAME=$(yq '.metadata.name' "$f")

if [ -n "$KIND" ]
then
    NAME="$NAME-$KIND"
fi 
NAME="$NAME.yaml"

echo "Working with  $NAME"

cp "$f" "$MANCACHE/$NAME"
done

# update webhook (add config.url and remove config.caBundle and config.service)
yq '.webhooks[0].clientConfig.url = "https://replace-url.com"' -i "$MANCACHE/kudo-manager-instance-admission-webhook-config.yaml"
yq 'del(.webhooks[0].clientConfig.caBundle)' -i "$MANCACHE/kudo-manager-instance-admission-webhook-config.yaml"
yq 'del(.webhooks[0].clientConfig.service)' -i "$MANCACHE/kudo-manager-instance-admission-webhook-config.yaml"

rm -rf "$TEMP"
echo "Finished"