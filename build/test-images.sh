#!/bin/sh

docker buildx build --build-arg GOLANG_VERSION=1.18 --platform linux/amd64,linux/arm64,linux/arm/v7 -t kudobuilder/golang:1.18 . --push

docker buildx build --build-arg GOLANG_VERSION=latest --platform linux/amd64,linux/arm64,linux/arm/v7 -t kudobuilder/golang:latest . --push
