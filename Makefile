OPERATOR_NAME := kops-autoscaler-openstack
IMAGE := elisaoyj/$(OPERATOR_NAME)

.PHONY: copybindata bindata clean deps test gofmt ensure check build build-image build-linux-amd64

copybindata:
	cp ${GOPATH}/src/k8s.io/kops/upup/models/bindata.go replace/bindata.go

clean:
	git clean -Xdf

bindata:
	cp replace/bindata.go vendor/k8s.io/kops/upup/models

deps:
	go get -u golang.org/x/lint/golint
	go get -u github.com/golang/dep/cmd/dep

test: bindata
	go test ./... -v -coverprofile=gotest-coverage.out > gotest-report.out && cat gotest-report.out || (cat gotest-report.out; exit 1)
	golint -set_exit_status cmd/... pkg/... > golint-report.out && cat golint-report.out || (cat golint-report.out; exit 1)
	./hack/gofmt.sh

gofmt:
	./hack/gofmt.sh

check:
	dep check|grep "lock is out of sync"; test $$? -eq 1

ensure:
	dep ensure -v
	cp replace/bindata.go vendor/k8s.io/kops/upup/models

build-linux-amd64: bindata
	rm -f bin/linux/$(OPERATOR_NAME)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -i -o bin/linux/$(OPERATOR_NAME) ./cmd

build: bindata
	rm -f bin/$(OPERATOR_NAME)
	go build -v -i -o bin/$(OPERATOR_NAME) ./cmd

build-image: build-linux-amd64
	docker build -t $(IMAGE):latest .
