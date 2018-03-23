#!/bin/bash

TAG="0.1.0"
IMG_NAME=spoter-controller:${TAG}

ORIGINDIR=$(pwd)

# build image
cp ./release/spoter-controller ./dockerfiles/spoter/
cd ./dockerfiles/spoter/
docker build -t ${IMG_NAME} .
rm ./spoter-controller
cd ${ORIGINDIR}
