#!/bin/bash

set -eux

# the tools we need
RIEMANNC=/var/vcap/jobs/riemannc/bin/riemannc
AWSCLI=/var/vcap/packages/awslogs/bin/aws
ETCDCTL=/var/vcap/packages/etcd/etcdctl

# location of etcd data store
ETCD_DATA_DIR=/var/vcap/store/etcd

# where to back up to
BACKUP_DIR=$(mktemp -d)

# where to compress backup to
ARCHIVE=$(mktemp)

# only backup if we are the leader, so exit if we are not
curl -s http://localhost:4001/v2/stats/leader | grep "not current leader" && exit 0;

# do the backup
${ETCDCTL} backup --data-dir ${ETCD_DATA_DIR} --backup-dir ${BACKUP_DIR}

# compress it
tar -czvf ${ARCHIVE} -C $BACKUP_DIR .

# copy it into s3
export LD_LIBRARY_PATH=/var/vcap/packages/awslogs/lib
${AWSCLI} s3 cp --sse AES256 ${ARCHIVE} s3://${S3_BUCKET_NAME}/$(date +%Y%m%d-%H%M).tar.gz

# clean up
rm ${ARCHIVE}
rm -r ${BACKUP_DIR}

# ping riemann
TTL=1800
${RIEMANNC} --service "kubernetes.etcd.backup" --host $(hostname) --ttl ${TTL} --metric_sint64 $(date +%s)
