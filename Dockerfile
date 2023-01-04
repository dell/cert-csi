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

FROM golang:1.16 as build-env

VOLUME [ "~/.kube/config", "/app/cert-csi/reports" ]

WORKDIR /app/cert-csi/
COPY go.mod go.sum ./
COPY ./certs/ /usr/local/share/ca-certificates/

ARG username
ARG password

RUN update-ca-certificates
RUN apt-get update && apt-get -qq -y install gridsite-clients
RUN git config --global credential.helper store && echo https://${username}:$(urlencode $password)@eos2git.cec.lab.emc.com > ~/.git-credentials

RUN GOSUMDB=off go mod download

COPY ./cmd/ ./cmd/
COPY ./pkg/ ./pkg/
COPY ./Makefile .

RUN make build

FROM frolvlad/alpine-glibc
WORKDIR /app/cert-csi/
COPY --from=build-env /app/cert-csi/cert-csi .

ENTRYPOINT [ "./cert-csi" ]
