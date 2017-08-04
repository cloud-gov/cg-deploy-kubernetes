#!/bin/bash


set -xue


# get all pods by service_id
curl -ks -u"${K8S_USERNAME}:${K8S_PASSWORD}" "${K8S_APISERVER}/api/v1/namespaces/default/pods/" | \
jq '.items[] | select( .metadata.name | test( "'"${SERVICE_ID}"'" ) ) | { name: .metadata.name, node: .status.hostIP, ip: .status.podIP }'

# run a few tests

# destroy server/0 via k8s API

sleep 5

# run a few tests

# destroy server/0 & server/0 via k8s API

sleep 10

# run a few tests

# destroy all servers via k8s API

sleep 20

# run a few tests

# Destroy a random sentinel

sleep 10

# run a few tests

# destroy a random proxy via k8s API

sleep 5

# run a few tests
