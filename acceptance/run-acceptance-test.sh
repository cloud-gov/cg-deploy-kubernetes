#!/bin/bash

set -e
set -u

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
  return 1
}

SERVICE_INSTANCE_NAME=$(mktemp "${SERVICE_NAME}-${PLAN_NAME}-XXXXXX")

cd ${TEST_PATH}

cf api $CF_API_URL
cf auth $CF_USERNAME $CF_PASSWORD

cf create-space -o $CF_ORGANIZATION $CF_SPACE
cf target -o $CF_ORGANIZATION -s $CF_SPACE

cf create-service ${SERVICE_NAME} ${PLAN_NAME} ${SERVICE_INSTANCE_NAME}

cf push --no-start -f ${MANIFEST_FILE:-manifest.yml}

if ! check_service; then
  echo "Failed to create service ${SERVICE_NAME}"
  exit 1
fi

cf bind-service ${APP_NAME} ${SERVICE_INSTANCE_NAME}
cf start ${APP_NAME}

cf delete -f ${APP_NAME}
cf delete-service -f ${SERVICE_INSTANCE_NAME}
