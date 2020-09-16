OPERATOR_NAME := kops-autoscaler-openstack
IMAGE := quay.io/elisaoyj/$(OPERATOR_NAME)

.PHONY: clean deps test gofmt ensure check build build-image build-linux-amd64 run

clean:
	git clean -Xdf

deps:
	GO111MODULE=off go get -u golang.org/x/lint/golint

test:
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

build-linux-amd64:
	rm -f bin/linux/$(OPERATOR_NAME)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build -mod vendor -v -i -o bin/linux/$(OPERATOR_NAME) ./cmd

build:
	rm -f bin/$(OPERATOR_NAME)
	GO111MODULE=on go build -mod vendor -v -i -o bin/$(OPERATOR_NAME) ./cmd

build-image: build-linux-amd64
	docker build -t $(IMAGE):latest .

run: build
	./bin/$(OPERATOR_NAME) --sleep 10
