#!/bin/bash

set -eux

cf api ${CF_API_URL}
(set +x; cf auth $CF_USERNAME $CF_PASSWORD)

cf target -o ${CF_ORGANIZATION}
if cf space ${CF_SPACE}; then
  cf delete-space -f ${CF_SPACE}
fi
