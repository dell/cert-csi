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
