#!/bin/bash

set -e
set -u

function cleanup() {
  cf delete -f ${APP_NAME}
  cf delete-service -f ${SERVICE_INSTANCE_NAME}
}

function check_service() {
  counter=36
  until [ $counter -le 0 ]; do
    status=$(cf service ${SERVICE_INSTANCE_NAME})
    if echo ${status} | grep "Status: create succeeded"; then
      return 0
    elif echo ${status} | grep "Status: create failed"; then
      return 1
    fi
    let counter-=1
    sleep 5
  done
}

cd ${TEST_PATH}

cf auth $CF_USERNAME $CF_PASSWORD

cf target -o $CF_ORGANIZATION 

cf create-space $CF_SPACE

cf target -s $CF_SPACE

cleanup

cf create-service ${SERVICE_NAME} ${PLAN_NAME} ${SERVICE_INSTANCE_NAME}

cf push --no-start -f ${MANIFEST_FILE:-manifest.yml}

if ! check_service; then
  echo "Failed to create service ${SERVICE_NAME}"
  exit 1
fi

cf bind-service ${APP_NAME} ${SERVICE_INSTANCE_NAME}
cf start ${APP_NAME}

cleanup
