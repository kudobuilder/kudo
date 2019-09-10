#!/bin/sh

# Pull the builder image with retries if it doesn't already exist.
retries=0
builder_image=$(grep FROM test/Dockerfile | awk '{print $2}')

if ! docker inspect "$builder_image"; then
    until docker pull "$builder_image"; do
        if [ retries -eq 3 ]; then
            echo "Giving up downloading builder image, failing build."
            exit 1
        fi
        echo "Docker pull failed, retrying."
        ((retries++))
        sleep 1
    done
fi

docker build -f test/Dockerfile -t kudo-test .
if [ $? -eq 0 ]
then
   docker run -it -m 4g -v $(pwd)/reports:/go/src/github.com/kudobuilder/kudo/reports --rm kudo-test
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
