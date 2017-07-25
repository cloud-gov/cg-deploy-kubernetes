#!/bin/bash
set -eux

# show cluster health
curl -v "https://${url}/health"
