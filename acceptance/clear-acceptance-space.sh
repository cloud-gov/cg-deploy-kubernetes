#!/bin/bash

set -eux

cf api ${CF_API_URL}
(set +x; cf auth $CF_USERNAME $CF_PASSWORD)

cf delete-space -f -o ${CF_ORGANIZATION} ${CF_SPACE}
