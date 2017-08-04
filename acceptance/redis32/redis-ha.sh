#!/bin/bash


set -xue


# get all pods by service_id
curl -ks -u"${K8S_USERNAME}:${K8S_PASSWORD}" "${K8S_APISERVER}/api/v1/namespaces/default/pods/" | \
jq '.items[] | select( .metadata.name | test( "'"${SERVICE_ID}"'" ) ) | { name: .metadata.name, node: .status.hostIP, ip: .status.podIP }'

# check proxy for address change.
# TODO: Will propbably do this with the sentinel logs rather than the proxy logs
# TODO: Better yet, the acceptance test app should actually make it easy to query this data from the sentinels I think.
curl -ks -u"${K8S_USERNAME}:${K8S_PASSWORD}" "${K8S_APISERVER}/api/v1/namespaces/default/pods/${POD_NAME}/log" | \
grep 'Master Address changed' | tail

# delete a pod
curl -ks -u"${K8S_USERNAME}:${K8S_PASSWORD}" "${K8S_APISERVER}/api/v1/namespaces/default/pods/${POD_NAME}" -XDELETE


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
