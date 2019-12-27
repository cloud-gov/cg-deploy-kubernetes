#!/bin/sh

set -eux

cd kubernetes-broker-exporter

go get -v ./...

go get -v github.com/onsi/ginkgo/ginkgo
ginkgo -r
