#!/bin/bash

set -e

go get -v github.com/square/certstrap
RED='\033[0;31m'
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
PURPLE='\033[0;35m'
NC='\033[0m'

if [[ -n $1 && $1 =~ (-h|--help)$ ]]
then
  echo "
  ./generate-k8s-release-certs.sh [--help, -h] [--grab-cert, -g] [<ca-cert> <ca-private-key>]

  For generating the Consul & K8s certificates and private keys based on a
  'single' root CA certificate for the 18F/cg-deploy-kubernetes.
  "
  exit
fi

local_ca_cert_name='consul_ca'
depot_path="k8s-certs"

if [[ -z $SAN_IPS ]]
then
  echo -e "${RED}ERROR${NC} Please set a ${YELLOW}\$SAN_IPS${NC} variable containing a comma-separated list of Kubernetes cluster member IPs"
  exit 97
fi

mkdir -p "${depot_path}"

if [[ -n $1 && $1 =~ (-g|--grab-cert)$ ]]
then
  if [[ -z $CG_PIPELINE ]]
  then
    echo -e "${RED}ERROR${NC} Please set a ${YELLOW}\$CG_PIPELINE${NC} variable pointing to a clone of ${YELLOW}https://github.com/18F/cg-pipeline-tasks${NC}"
    echo -e "eg, ${PURPLE}CG_PIPELINE=~/dev/cg-pipeline-tasks ./generate-k8s-release-certs.sh --grab-cert${NC}"
    exit 98
  fi

  if [[ -z "${ci_env}" ]]
  then
    echo -e "${RED}ERROR${NC} Please set a ${YELLOW}\$ci_env${NC} variable to continue from ${YELLOW}fly targets${NC}"
    echo -e "eg, ${PURPLE}ci_env=fr ./generate-k8s-release-certs.sh --grab-cert${NC}"
    exit 99
  fi

  # Download deploy-bosh pipeline
  deploy_bosh_json=$(
  fly --target "${ci_env}" \
      get-pipeline \
      --pipeline deploy-bosh | \
  spruce json
  )

  echo -e "${GREEN}Downloading${NC} master-bosh-root-cert"
  eval "$(
  echo "${deploy_bosh_json}" | \
  jq -r '
    .resources[] |
    select( .name == "master-bosh-root-cert" ) |
    @sh "export AWS_DEFAULT_REGION=\(.source.region_name)
    export CA_CERT=\(.source.versioned_file)
    export CA_BUCKET=\(.source.bucket)"
  '
  )"
  aws s3 cp "s3://${CA_BUCKET}/${CA_CERT}" "${depot_path}/${CA_CERT}"

  echo -e "${GREEN}Downloading${NC} master-bosh-root-key"
  eval "$(
  echo "${deploy_bosh_json}" | \
  jq -r '
    .resources[] |
    select( .name == "common-masterbosh" ) |
    @sh "export AWS_DEFAULT_REGION=\(.source.region)
    export PASSPHRASE=\(.source.secrets_passphrase)
    export CA_KEY_ENCRYPTED=\(.source.bosh_cert)
    export CA_BUCKET=\(.source.bucket_name)"
  '
  )"
  aws s3 cp "s3://${CA_BUCKET}/${CA_KEY_ENCRYPTED}" "${depot_path}/${CA_KEY_ENCRYPTED}"

  echo -e "${GREEN}Decrypting${NC} master-bosh-root-key"
  export INPUT_FILE="${depot_path}/${CA_KEY_ENCRYPTED}"
  export OUTPUT_FILE=$(
    echo "${depot_path}/${CA_KEY_ENCRYPTED}" | \
    sed 's/\.pem/.key/'
  )
  eval "$(
  echo "${deploy_bosh_json}" | \
  jq -r '
    .jobs[] |
    select( .name == "deploy-master-bosh" ) |
    .plan[] |
    select( .task == "decrypt-private-key") |
    @sh "export PASSPHRASE=\(.params.PASSPHRASE)"
  '
  )"
  "${CG_PIPELINE}"/decrypt.sh
  local_ca_cert_name="${CA_KEY_ENCRYPTED//\.pem/}"
  echo -e "${GREEN}Signing${NC} certificates with certificate authority ${YELLOW}${local_ca_cert_name}${NC} and key"
elif [[ -n $1 && -n $2 ]]
then
  echo -e "${YELLOW}Copying ${1},${2} to ${depot_path}${NC}"
  cp -p {"$1","$2"} "${depot_path}"/.
  local_ca_cert_name=$(basename "$1" | sed 's/\.crt//')
  echo -e "${GREEN}Signing${NC} certificates with supplied certificate authority ${YELLOW}${local_ca_cert_name}${NC} and key"
else
  echo -e "${GREEN}Creating${NC} ${YELLOW}new${NC} certificate authority ${YELLOW}${local_ca_cert_name}${NC} and key"
  certstrap --depot-path "${depot_path}" init --passphrase '' --common-name "${local_ca_cert_name}"
fi

echo -e "${CYAN}Generating${NC} Consul key and certificate pairs"


# Server certificate to share across the consul cluster
# You need the IP SAN for actions performed on the localhost by bosh/monit (like consul leave)
consul_server_cn=server.dc1.cf.internal
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name "${consul_server_cn}" --ip 127.0.0.1
certstrap --depot-path ${depot_path} sign "${consul_server_cn}" --CA "${local_ca_cert_name}"
mv -f ${depot_path}/$consul_server_cn.key ${depot_path}/consul_server.key
mv -f ${depot_path}/$consul_server_cn.csr ${depot_path}/consul_server.csr
mv -f ${depot_path}/$consul_server_cn.crt ${depot_path}/consul_server.crt

# Agent certificate to distribute to jobs that access consul
certstrap --depot-path ${depot_path} request-cert --passphrase '' --common-name 'consul agent'
certstrap --depot-path ${depot_path} sign 'consul_agent' --CA "${local_ca_cert_name}"
mv -f ${depot_path}/consul_agent.key ${depot_path}/consul_agent.key
mv -f ${depot_path}/consul_agent.csr ${depot_path}/consul_agent.csr
mv -f ${depot_path}/consul_agent.crt ${depot_path}/consul_agent.crt


echo -e "${CYAN}Generating${NC} Kubernetes key and certificate pairs"
certstrap --depot-path ${depot_path} request-cert --passphrase '' --cn kubernetes --domain kubernetes.default.svc.cluster.local,kubernetes.default.svc --ip "${SAN_IPS}",127.0.0.1,10.0.0.1
certstrap --depot-path ${depot_path} sign 'kubernetes' --CA "${local_ca_cert_name}"
