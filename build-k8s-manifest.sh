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

export SPRUCE_FILE_BASE_PATH=./kubernetes-config
spruce merge \
  --prune meta \
  --prune terraform_outputs \
  kubernetes-release/templates/k8s-deployment.yml \
  kubernetes-release/templates/k8s-jobs.yml \
  kubernetes-release/templates/k8s-infrastructure-aws.yml \
  common-secret/${TARGET_ENVIRONMENT}-kubernetes.yml \
  kubernetes-config/k8s-jobs.yml \
  kubernetes-config/infrastructure-aws-${TARGET_ENVIRONMENT}.yml \
  terraform-yaml/state.yml \
  > kubernetes-manifest/manifest.yml
