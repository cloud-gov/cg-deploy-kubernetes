#!/bin/bash

set -xue

# Test Redis is working by running `SET`, `GET`, and `DEL`.
run_tests() {
  if ! $(curl -kfv "https://${url}/")
  then
    echo "error with testing Redis."
    exit 99
  fi
}

# Get all pods from Kubernetes matching $idx_and_short_serviceid
get_k8s_pods() {
  curl -G -kfv -u"${K8S_USERNAME}:${K8S_PASSWORD}" "${K8S_APISERVER}/api/v1/namespaces/default/pods?labelSelector=idx_and_short_serviceid%3D${idx_and_short_serviceid}" | \
  jq -re '.items[] | { name: .metadata.name, node: .status.hostIP, ip: .status.podIP, status: .status.phase }' | \
  jq -s '.'
}

# Get the current primary server's IP address
get_primary_ip() {
  curl -kfv "https://${url}/config-get?p=slave-announce-ip" | jq -re '.["slave-announce-ip"]'
}

# Get the current primary server's role
get_primary_role() {
  curl -kfv "https://${url}/info?s=replication" | jq -re ".role"
}

# Get the number of replicas that the primary server knows about
get_replica_count() {
  curl -kfv "https://${url}/info?s=replication" | jq -re '.connected_slaves'
}

# Iterate on number of replicas to verify that we're at 3x servers
check_number_of_replicas() {
  counter=120
  until [ $counter -le 0 ]
  do
    if [[ $(get_primary_role) != "master" ]]
    then
      let counter-=1
      sleep 5
    fi
    if [ $(($(get_replica_count) + 0)) -lt $replica_count ]
    then
      let counter-=1
      sleep 5
    else
      return 0
    fi
  done
  return 1
}

export replica_count=$(($(get_replica_count) + 0))

run_tests

primary_server_ip=$(get_primary_ip)

primary_server_name=$(
  echo "$(get_k8s_pods)" | \
  jq -re '.[] | select( .ip == "'"${primary_server_ip}"'") | .name'
)

# Check to see if there were any errors retrieving $primary_server_name
if ! echo "${primary_server_name}" | grep -oE 'x[a-zA-Z0-9]{3,15}' > /dev/null
then
  echo "There was an error getting the primary server's name for ${primary_server_ip}"
  exit 1
fi

# Delete the current master
curl -kv -u"${K8S_USERNAME}:${K8S_PASSWORD}" \
  "${K8S_APISERVER}/api/v1/namespaces/default/pods/${primary_server_name}" \
  -XDELETE

if ! check_number_of_replicas
then
  echo "Number of servers never hit 3x or the proxy dropped the connection"
  echo "redis-cli INFO"
  curl -kfv "https://${url}/info"
  echo "redis-cli CONFIG GET *"
  curl -kfv "https://${url}/config-get"
  exit 1
fi

# Allow for the proxies to find the new master.
sleep 1

run_tests
