OPERATOR_NAME := kops-autoscaler-openstack
IMAGE := quay.io/elisaoyj/$(OPERATOR_NAME)

.PHONY: clean deps test gofmt ensure check build build-image build-linux-amd64 run

clean:
	git clean -Xdf

deps:
	go get -u golang.org/x/lint/golint

test:
	go test ./... -v -coverprofile=gotest-coverage.out > gotest-report.out && cat gotest-report.out || (cat gotest-report.out; exit 1)
	golint -set_exit_status cmd/... pkg/... > golint-report.out && cat golint-report.out || (cat golint-report.out; exit 1)
	go vet ./...
	./hack/gofmt.sh
	git diff --exit-code go.mod go.sum

gofmt:
	./hack/gofmt.sh

ensure:
	go mod tidy

build-linux-amd64:
	rm -f bin/linux/$(OPERATOR_NAME)
	go version
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build -v -o bin/linux/$(OPERATOR_NAME) ./cmd

build:
	rm -f bin/$(OPERATOR_NAME)
	go build -v -o bin/$(OPERATOR_NAME) ./cmd

build-image: build-linux-amd64
	docker build -t $(IMAGE):latest .

run: build
	./bin/$(OPERATOR_NAME) --loglevel 4 --sleep 10
