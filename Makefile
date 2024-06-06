#
#
# Copyright © 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#      http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#

# determine if podman or docker should be used (use podman if found)
ifneq (, $(shell which podman 2>/dev/null))
export BUILDER=podman
else
export BUILDER=docker
endif

.PHONY: clean check gosec gocyclo test gocover build distribute install-ms

all: clean build

clean:
	go clean -r -x ./...

check:	gosec
	gofmt -w ./.
	golint -set_exit_status ./...
	go vet

gosec:
	gosec -quiet -log gosec.log -out=gosecresults.csv -fmt=csv ./...

gocyclo:
	gocyclo -top 10 .
	@gocyclo -avg . |grep Average

test:
	(go test -race -v -coverprofile=c.out ./...)

gocover:
	go tool cover -html=c.out

build:
	go build -ldflags "-s -w" ./cmd/cert-csi

docker: download-csm-common
	$(eval include csm-common.mk)
	@echo "Building base image from $(DEFAULT_BASEIMAGE) and loading dependencies..."
	./scripts/build_ubi_micro.sh $(DEFAULT_BASEIMAGE)
	@echo "Base image build: SUCCESS"
	$(eval BASEIMAGE=localhost/cert-ubimicro:latest)
	$(BUILDER) build -t cert-csi:latest --build-arg BASEIMAGE=$(BASEIMAGE) --build-arg GOIMAGE=$(DEFAULT_GOIMAGE) .

# build-statically-linked : used for building a standalone binary with statically linked glibc
# this command should be used when building the binary for distributing it to customer/user
build-statically-linked:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CGO_LDFLAGS="-static" go build ./cmd/cert-csi

install-nix:
	go build -ldflags "-s -w" ./cmd/cert-csi
	./autocomplete.sh

distribute:
	GOOS=windows GOARCH=amd64 go build -o build/cert-csi.exe -ldflags "-s -w" ./cmd/cert-csi
	GOOS=linux GOARCH=amd64 go build -o build/cert-csi -ldflags "-s -w" ./cmd/cert-csi

install-ms:
	kubectl create -f ./pkg/k8sclient/manifests/metrics-server

delete-ms:
	kubectl delete -f ./pkg/k8sclient/manifests/metrics-server

download-csm-common:
	curl -O -L https://raw.githubusercontent.com/dell/csm/main/config/csm-common.mk
