#!/bin/bash
set -e
set -u

cat << EOF > ./cloudwatch-params.yaml
${CLOUDWATCH_PARAMS}
EOF

spruce merge --prune meta kubernetes-config/manifests/fluentd-cloudwatch-template.yaml \
  ./cloudwatch-params.yaml \
  > kubernetes-config/manifests/fluentd-cloudwatch.yaml

cat << EOF > ./kube2iam-params.yaml
${KUBE2IAM_PARAMS}
EOF
spruce merge --prune meta kubernetes-config/manifests/kube2iam-template.yaml \
  ./kube2iam-params.yaml \
  > kubernetes-config/manifests/kube2iam.yaml

cat << EOF > ./riemann-podstatus.yaml
${RIEMANN_PODSTATUS_PARAMS}
EOF
spruce merge --prune meta kubernetes-config/manifests/riemann-podstatus-template.yaml \
  ./riemann-podstatus.yaml \
  > kubernetes-config/manifests/riemann-podstatus.yaml


export SPRUCE_FILE_BASE_PATH=./kubernetes-config
kubernetes-release/generate_deployment_manifest \
  common-secret/secrets.yml \
  kubernetes-config/k8s-jobs.yml \
  kubernetes-config/infrastructure-aws-${TARGET_ENVIRONMENT}.yml \
  > kubernetes-manifest/manifest.yml
