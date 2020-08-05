#!/usr/bin/env bash

if ! command -v curl &> /dev/null
then
    echo "curl command NOT found"
    exit 1
fi


CODE=$(curl -s -o /dev/null -w "%{http_code}" localhost:4040)


case "$CODE" in 
    "000") 
        echo "ngrok server is not up."
        exit 1;;

    "302") echo "ngrok up and running" ;;
esac

if ! command -v yq &> /dev/null
then
    echo "yq command NOT found"
    exit 1
fi

#  need to update the webhook url to the current tunnel.  ngrok array order changes requiring a tunnel select for https and pull the public_url of that tunnel
#  template webfile located at: config/admit-wh.yaml relative to the project root
#  this script requires "update-manifests.sh" to run first
yq  w hack/manifest-gen/kudo-manager-instance-admission-webhook-config.yaml webhooks[0].clientConfig.url "$(curl -s localhost:4040/api/tunnels | jq '.tunnels[] | select(.proto == "https") | .public_url' -r)/admit-kudo-dev-v1beta1-instance" | kubectl apply -f -

# debug notes:  This kubectl apply will fail if a kudo init creates a webhook with a clientConfig.service.  By adding  a clientConfig.url it creates a service and url which is not valid
RESULT=$?
if [ "$RESULT" -ne 0 ]; then 
    echo "updating the webhook config FAILED.  It is likely because the deploy configuration includes a service which is incompatible with a config url."
    echo "you may need to 'kubectl delete MutatingWebhookConfiguration kudo-manager-instance-admission-webhook-config'"
fi