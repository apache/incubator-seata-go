GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GO_VERSION = 1.15

REGISTRY ?= seataio
TAG = $(shell git rev-parse --abbrev-ref HEAD)
COMMIT = git-$(shell git rev-parse --short HEAD)
DATE = $(shell date +"%Y-%m-%d_%H:%M:%S")
GOLDFLAGS = "-w -s -extldflags '-z now' -X github.com/opentrx/seata-golang/versions.COMMIT=$(COMMIT) -X github.com/opentrx/seata-golang/versions.BUILDDATE=$(DATE)"


.PHONY: build-go
build-go:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildmode=pie -o $(CURDIR)/dist/tc -ldflags $(GOLDFLAGS) -v ./cmd/tc

.PHONY: build-bin
build-bin:
	docker run --rm -e GOOS=linux -e GOCACHE=/tmp -e GOARCH=$(ARCH) -e GOPROXY=https://goproxy.cn \
		-u $(shell id -u):$(shell id -g) \
		-v $(CURDIR):/go/src/github.com/opentrx/seata-golang:ro \
		-v $(CURDIR)/dist:/go/src/github.com/opentrx/seata-golang/dist/ \
		golang:$(GO_VERSION) /bin/bash -c '\
		cd /go/src/github.com/opentrx/seata-golang && \
		make build-go '

.PHONY: build-images
build-images: build-bin
	docker build -t $(REGISTRY)/seata-golang:$(TAG) --build-arg ARCH=amd64 -f dist/Dockerfile dist/

.PHONY: push
push:
	docker push $(REGISTRY)/seata-golang:$(TAG)

.PHONY: lint
lint:
	@gofmt -d $(GOFILES_NOVENDOR)
	@if [ $$(gofmt -l $(GOFILES_NOVENDOR) | wc -l) -ne 0 ]; then \
		echo "Code differs from gofmt's style" 1>&2 && exit 1; \
	fi
	@GOOS=linux go vet ./...
	@GOOS=linux gosec -exclude=G204,G601 ./...


.PHONY: clean
clean:
	$(RM) dist/tc