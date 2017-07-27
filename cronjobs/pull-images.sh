#!/bin/bash

set -eux

export PATH=$PATH:/var/vcap/packages/docker/bin

for image in $(echo "${IMAGES}"); do
  docker \
    --host unix:///var/vcap/sys/run/docker/docker.sock \
    pull "${image}"
done
