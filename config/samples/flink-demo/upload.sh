#!/bin/bash

set -e

wget -O job.jar ${DOWNLOAD_URL}
curl -X POST -H "Expect:" -F "jarfile=@job.jar" http://${JOBMANAGER}:8081/jars/upload | jq .
sleep 5
jarid=`curl http://${JOBMANAGER}:8081/jars?name=job.jar | jq -r .files[0].id`
sleep 5
jobid=`curl -XPOST -d '{}' http://${JOBMANAGER}:8081/jars/$jarid/run | jq .`

echo "Running job $jobid"