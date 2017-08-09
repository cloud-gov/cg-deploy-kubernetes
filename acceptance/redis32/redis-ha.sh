#!/bin/bash

set -xe

# Test Redis is working by running `SET`, `GET`, and `DEL`.
run_tests() {
  if ! $(curl -kfs "https://${url}/")
  then
    echo "error with testing Redis."
    exit 99
  fi
}

# Get all pods from Kubernetes matching $idx_and_short_serviceid
get_k8s_pods() {
  curl -ksf -u"${K8S_USERNAME}:${K8S_PASSWORD}" "${K8S_APISERVER}api/v1/namespaces/default/pods?labelSelector=idx_and_short_serviceid%3D${idx_and_short_serviceid}" | \
  jq '.items[] | { name: .metadata.name, node: .status.hostIP, ip: .status.podIP, status: .status.phase }' \ |
  jq -s '.'
}

# Get the current primary server's IP address
get_primary_ip() {
  curl -kfs "https://${url}/config-get?p=slave-announce-ip"
}

# Get the current primary server's role
get_primary_role() {
  curl -kfs "https://${url}/info?s=replication" | \
  jq -re ".role"
}

# Get the number of replicas that the primary server knows about
get_replica_count() {
  curl -kfs "https://${url}/info?s=replication" | \
  jq -re '.connected_slaves'
}

# Iterate on number of replicas to verify that we're at 3x servers
check_number_of_replicas() {
  counter=120
  until [ $counter -le 0 ]
  do
    #if [[ $(get_primary_role) != "master" ]]
    #then
      #echo "The proxy isn't connected to the master. This shouldn't happen"
      #return 1
    #fi
    if [ $(get_replica_count) -lt 2 ]
    then
      let counter-=1
      sleep 5
    else
      return 0
    fi
  done
  return 1
}

run_tests

primary_server_ip=$(get_primary_ip)

primary_server_name=$(
  echo "$(get_k8s_pods)" | \
  jq '.[] | select( .ip == "'${primary_server_ip}'") | .name'
)

# Delete the current master
curl -ks -u"${K8S_USERNAME}:${K8S_PASSWORD}" \
  "${K8S_APISERVER}/api/v1/namespaces/default/pods/${primary_server_name}" \
  -XDELETE

if ! check_number_of_replicas
then
  echo "Number of servers never hit 3x"
  curl -kv "https://${url}/info"
  curl -kv "https://${url}/config-get"
  exit 1
fi

run_tests
