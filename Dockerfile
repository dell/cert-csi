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
