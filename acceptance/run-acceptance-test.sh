#!/bin/bash

set -e
set -u

function cleanup() {
  cf delete -f ${APP_NAME}
  cf delete-service -f ${SERVICE_INSTANCE_NAME}
}

cd ${TEST_PATH}

cf login \
  -a $CF_API_URL \
  -u $CF_USERNAME \
  -p $CF_PASSWORD \
  -o $CF_ORGANIZATION \
  -s $CF_SPACE

cleanup

cf create-service ${SERVICE_NAME} ${PLAN_NAME} ${SERVICE_INSTANCE_NAME}

cf push --no-start -f ${MANIFEST_FILE:-manifest.yml}
cf bind-service ${APP_NAME} ${SERVICE_INSTANCE_NAME}
cf start ${APP_NAME}

cleanup
