#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
TOP_DIR="${DIR}/.."

BUILD_CONTAINER_FOCAL="tomhp.fritz.box:5555/docker/pbuilder-container-focal:v1.5.1"

mkdir -p ${TOP_DIR}/Build/pbuilder/focal
docker run \
      --workdir=/home/build-user/s \
      --rm \
      --privileged \
      -v /proc:/proc \
      -v $TOP_DIR:/home/build-user/s \
      -v $TOP_DIR/Build/pbuilder/focal:/var/cache/pbuilder/result \
      $BUILD_CONTAINER_FOCAL \
      build_signed_package.sh