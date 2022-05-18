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

REPO=github.com/ocgi/carrier-sdk
REGISTRY_NAME=ocgi
GIT_COMMIT=$(shell git rev-parse "HEAD^{commit}")
VERSION=$(shell git describe --tags --abbrev=14 "${GIT_COMMIT}^{commit}" --always)

OUTDIR=${PWD}/bin


all: fmt vet sdk-server image push

clean: ## make clean
	rm -rf ${OUTDIR}

fmt:
	@echo "run go fmt ..."
	@go fmt ./sdks/sdkgo/...
	@go fmt ./pkg/...

vet:
	@echo "run go vet ..."
	@go vet ./sdks/sdkgo/...
	@go vet ./pkg/...

sdk-server:
	@echo "build sdk-server"
	cd cmd/sdk-server && CGO_ENABLED=0 GOOS=linux go build -o ${OUTDIR}/sdk-server

image: sdk-server
	docker build -t $(REGISTRY_NAME)/carrier-sdkserver:$(VERSION) -f cmd/sdk-server/Dockerfile .

push:	image
	docker push $(REGISTRY_NAME)/carrier-sdkserver:$(VERSION)

test-sdk-go:
	go test $(REPO)/sdks/sdkgo/...

test-pkg:
	go test $(REPO)/pkg/...

test: test-pkg test-sdk-go
