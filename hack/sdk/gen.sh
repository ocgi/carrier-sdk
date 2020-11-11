#!/bin/bash

# Copyright 2021 The OCGI Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

PROG_PATH=${BASH_SOURCE[0]}
BASE_DIR="$(cd "$(dirname "${PROG_PATH:-$PWD}")/../.." 2>/dev/null 1>&2 && pwd)"

header() {
    cat ${BASE_DIR}/hack/boilerplate.go.txt "$1" | sponge "$1"
}

sdk=${BASE_DIR}/api/proto/v1alpha1
googleapis=${BASE_DIR}/api/proto/googleapis

export GO111MODULE=on


cd ${BASE_DIR}


# generate the go code and grpc gateway for each feature stage
protoc --experimental_allow_proto3_optional \
        -I ${googleapis} \
        -I ${sdk} \
        --go_out=sdks/sdkgo/api/v1alpha1 \
        --go-grpc_out=sdks/sdkgo/api/v1alpha1 \
        --grpc-gateway_out=sdks/sdkgo/api/v1alpha1 \
        --grpc-gateway_opt logtostderr=true \
        sdk.proto

#protoc --experimental_allow_proto3_optional -I ${googleapis} -I ${sdk} sdk.proto --go_out=sdks/sdkgo/api --go-grpc_out=sdks/sdkgo/api

# generate the cpp code
protoc --experimental_allow_proto3_optional \
        -I ${googleapis} \
        -I ${sdk} \
        --cpp_out=sdks/sdkcpp/api/v1alpha1 \
        --grpc_out=sdks/sdkcpp/api/v1alpha1 \
        --plugin=protoc-gen-grpc=`which grpc_cpp_plugin` \
        sdk.proto google/api/annotations.proto google/api/http.proto
