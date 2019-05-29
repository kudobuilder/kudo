#!/bin/sh


docker build --build-arg GOLANG_VERSION=1.10 -t kudobuilder/golang:1.10 .
docker push kudobuilder/golang:1.10

docker build --build-arg GOLANG_VERSION=1.11 -t kudobuilder/golang:1.11 .
docker push kudobuilder/golang:1.11

docker build --build-arg GOLANG_VERSION=1.12 -t kudobuilder/golang:1.12 .
docker push kudobuilder/golang:1.12

docker build --build-arg GOLANG_VERSION=latest -t kudobuilder/golang:latest .
docker push kudobuilder/golang:latest