#!/bin/bash
# 
# This script cleans up unused disk space in docker
# 

set -eux

/var/vcap/packages/docker/bin/docker \
	--host unix:///var/vcap/sys/run/docker/docker.sock \
	rmi $(/var/vcap/packages/docker/bin/docker \
		--host unix:///var/vcap/sys/run/docker/docker.sock \
		images -q --filter "dangling=true")
