#!/bin/bash

set -e
set -u
set -x

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

INSTANCE_NAME=$(mktemp "${SERVICE_NAME}-${PLAN_NAME}-XXXXXX")
SERVICE_INSTANCE_NAME="${INSTANCE_NAME}"
APP_NAME="${INSTANCE_NAME}"

cd ${TEST_PATH}

cf api ${CF_API_URL}
(set +x; cf auth $CF_USERNAME $CF_PASSWORD)

cf create-space -o ${CF_ORGANIZATION} ${CF_SPACE}
cf target -o ${CF_ORGANIZATION} -s ${CF_SPACE}

cf create-service ${SERVICE_NAME} ${PLAN_NAME} ${SERVICE_INSTANCE_NAME}

cf push --no-start -f ${MANIFEST_FILE:-manifest.yml} ${APP_NAME}

if ! check_service; then
  echo "Failed to create service ${SERVICE_NAME}"
  exit 1
fi

cf bind-service ${APP_NAME} ${SERVICE_INSTANCE_NAME}
cf start ${APP_NAME}

url=$(cf app ${APP_NAME} | grep -e "urls: " -e "routes: " | awk '{print $2}')
status=$(curl -w "%{http_code}" "https://${url}")
if [ "${status}" != "200" ]; then
  echo "Unexpected status code ${status}"
  cf logs ${APP_NAME} --recent
  exit 1
fi

# show cluster health
set +e
curl -v "https://${url}/health"

cf delete -f ${APP_NAME}
cf delete-service -f ${SERVICE_INSTANCE_NAME}
