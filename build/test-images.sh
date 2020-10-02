#!/bin/sh

docker build --build-arg GOLANG_VERSION=1.15 -t kudobuilder/golang:1.15 .
docker push kudobuilder/golang:1.15

docker build --build-arg GOLANG_VERSION=latest -t kudobuilder/golang:latest .
docker push kudobuilder/golang:latest
