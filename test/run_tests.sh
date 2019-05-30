#!/bin/sh

if [ ! -f test/.git-credentials ]; then
    echo ".git-credentials file expected in the test directory for tests to be run. Please provide the file"
    exit 1
fi

docker build -f test/Dockerfile -t kudo-test .
if [ $? -eq 0 ]
then
   docker run kudo-test
else
    echo "Error when building test docker image, cannot run tests."
fi

DOCKER_EXIT_CODE=$?

if [ $DOCKER_EXIT_CODE -eq 0 ]
then
   echo "Tests finished successfully! ヽ(•‿•)ノ"
fi

exit $DOCKER_EXIT_CODE