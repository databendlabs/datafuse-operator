#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# corresponding to go mod init <module>
MODULE=datafuselabs.io/datafuse-operator
# api package
APIS_PKG=pkg/apis
# generated output package
OUTPUT_PKG=pkg/client
# group-version such as foo:v1alpha1
GROUP_VERSION=datafuse:v1alpha1

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}
mkdir -p ${SCRIPT_ROOT}/tmp
mkdir -p ${SCRIPT_ROOT}/pkg
mkdir -p ${SCRIPT_ROOT}/pkg/client
# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
bash "${CODEGEN_PKG}"/generate-groups.sh all \
  ${MODULE}/${OUTPUT_PKG} ${MODULE}/${APIS_PKG} \
  ${GROUP_VERSION} \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt \
  --output-base "${SCRIPT_ROOT}/tmp"
#  --output-base "${SCRIPT_ROOT}/../../.." \
mv ${SCRIPT_ROOT}/tmp/${MODULE}/${OUTPUT_PKG}/*  ${SCRIPT_ROOT}/pkg/client/
rm -rf ${SCRIPT_ROOT}/tmp