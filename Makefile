OPERATOR_NAME := kops-autoscaler-openstack
IMAGE := elisaoyj/$(OPERATOR_NAME)

.PHONY: copybindata bindata clean deps test gofmt ensure check build build-image build-linux-amd64 run

copybindata:
	cp ${GOPATH}/src/k8s.io/kops/upup/models/bindata.go replace/bindata.go

clean:
	git clean -Xdf

bindata:
	cp replace/bindata.go vendor/k8s.io/kops/upup/models

deps:
	go get -u golang.org/x/lint/golint

test: bindata
	GO111MODULE=on go test ./... -mod vendor -v -coverprofile=gotest-coverage.out > gotest-report.out && cat gotest-report.out || (cat gotest-report.out; exit 1)
	GO111MODULE=on golint -set_exit_status cmd/... pkg/... > golint-report.out && cat golint-report.out || (cat golint-report.out; exit 1)
	GO111MODULE=on go vet -mod vendor ./...
	./hack/gofmt.sh
	git diff --exit-code go.mod go.sum

gofmt:
	./hack/gofmt.sh

ensure:
	GO111MODULE=on go mod tidy
	GO111MODULE=on go mod vendor
	cp replace/bindata.go vendor/k8s.io/kops/upup/models

build-linux-amd64: bindata
	rm -f bin/linux/$(OPERATOR_NAME)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build -mod vendor -v -i -o bin/linux/$(OPERATOR_NAME) ./cmd

build: bindata
	rm -f bin/$(OPERATOR_NAME)
	GO111MODULE=on go build -mod vendor -v -i -o bin/$(OPERATOR_NAME) ./cmd

build-image: build-linux-amd64
	docker build -t $(IMAGE):latest .

run: build
	./bin/$(OPERATOR_NAME) --sleep 10
