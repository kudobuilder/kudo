#!/bin/sh

docker build -f test/Dockerfile -t kudo-test .
if [ $? -eq 0 ]
then
   docker run kudo-test
else
    echo "Error when building test docker image, cannot run tests."
    exit 1
fi

DOCKER_EXIT_CODE=$?

if [ $DOCKER_EXIT_CODE -eq 0 ]
then
   echo "Tests finished successfully! ヽ(•‿•)ノ"
fi

exit $DOCKER_EXIT_CODE