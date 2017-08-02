#!/bin/sh

set -eux

export GOPATH=$(pwd)/gopath
export PATH=$PATH:$GOPATH/bin

cd gopath/src/github.com/18F/kubernetes-broker-exporter

go get github.com/Masterminds/glide
go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega

glide install
ginkgo -r
