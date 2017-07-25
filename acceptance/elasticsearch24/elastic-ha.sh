#!/bin/bash
set -eux

cluster_status() {
	curl -s https://${url}/cluster-health | jq -r .status
}

cluster_master() {
	curl -s https://${url}/cluster-state | jq -r '.nodes[.master_node].name'
}

master_count() {
	curl -s https://${url}/cluster-health | jq '.number_of_nodes-.number_of_data_nodes'
}

data_count() {
	curl -s https://${url}/cluster-health | jq .number_of_data_nodes
}

check_status() {
	counter=120
	until [ $counter -le 0 ]; do
		if [ $(cluster_status) == "red" ]; then
			echo "Cluster status is red. This shouldn't happen."
			return 1
		fi
		if [ $(cluster_status) != "green" ]; then
			let counter-=1
			sleep 5
		else
			return 0
		fi
	done
	return 1
}

function check_new_master() {
	counter=120
	until [ $counter -le 0 ]; do
		if [ $(cluster_master) == "${ORIG_MASTER}" ]; then
			let counter-=1
			sleep 5
		else
			return 0
		fi
	done
	return 1
}

function check_master_count() {
	counter=120
	until [ $counter -le 0 ]; do
		if [ $(cluster_status) == "red" ]; then
			echo "Cluster went red during master heal"
			return 1
		fi
		if [ $(master_count) != "${NUM_MASTER}" ]; then
			let counter-=1
			sleep 5
		else
			return 0
		fi
	done
	return 1
}

function check_data_count() {
	counter=120
	until [ $counter -le 0 ]; do
		if [ $(cluster_status) == "red" ]; then
			echo "Cluster went red during data heal"
			return 1
		fi
		if [ $(data_count) != "${NUM_DATA}" ]; then
			let counter-=1
			sleep 5
		else
			return 0
		fi
	done
	return 1
}

# make sure we are green before starting

if ! check_status; then
	echo "Cluster didn't start green"
	curl -v https://${url}/cluster-health
	exit 1;
fi

# get our current master, and count
ORIG_MASTER=$(cluster_master)
NUM_MASTER=$(master_count)

# kill it
curl -ks -u"${K8S_USERNAME}:${K8S_PASSWORD}" ${K8S_APISERVER}/api/v1/namespaces/default/pods/${ORIG_MASTER} -XDELETE

# wait for it to die, i.e. we should see 1 less master
ORIG_NUM_MASTER=${NUM_MASTER}
NUM_MASTER=$((${NUM_MASTER}-1))
if ! check_master_count; then
	echo "Failed to simulate master node failure"
	curl -v https://${url}/cluster-health
	curl -v https://${url}/cluster-state
	exit 1;
fi
NUM_MASTER=${ORIG_NUM_MASTER}

# ensure we pick a new master and stay green
if ! check_new_master; then
  echo "Failed to elect new master"
  exit 1
fi

if ! check_status; then
	echo "Cluster didn't stay green during master election"
	curl -v https://${url}/cluster-health
	exit 1;
fi

if ! check_master_count; then
  echo "Failed to heal master"
  exit 1
fi

# remember how many data nodes we should have
NUM_DATA=$(data_count)

# get a random data node
DATA_NODE=$(curl -s https://${url}/cluster-state | jq -r '.nodes | map(select(.attributes.master=="false")) | .[].name' | shuf | head -1)

# kill it
curl -ks -u"${K8S_USERNAME}:${K8S_PASSWORD}" ${K8S_APISERVER}/api/v1/namespaces/default/pods/${DATA_NODE} -XDELETE

# wait for it to die, we should see one less data node
ORIG_NUM_DATA=${NUM_DATA}
NUM_DATA=$((${NUM_DATA}-1))
if ! check_data_count; then
	echo "Failed to simulate data node failure"
	curl -v https://${url}/cluster-health
	curl -v https://${url}/cluster-state
	exit 1;
fi
NUM_DATA=${ORIG_NUM_DATA}

# the data node should come back
if ! check_data_count; then
	echo "Cluster did not heal a lost data node"
	curl -v https://${url}/cluster-health
	curl -v https://${url}/cluster-state
	exit 1;
fi

# the cluster should be green
if ! check_status; then
	echo "Cluster didn't go green after healing a data node"
	curl -v https://${url}/cluster-health
	exit 1;
fi
