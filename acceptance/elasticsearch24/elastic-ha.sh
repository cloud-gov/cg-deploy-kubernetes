#!/bin/bash
set -eux

# show cluster health
curl -v "https://${url}/cluster-health"


# show cluster nodes
curl -v "https://${url}/cluster-nodes"

# mess with k8s
curl -kv -u'${K8S_USERNAME}:${K8S_PASSWORD}' ${K8S_APISERVER}

