AIRVER := $(shell git describe --tags)
LDFLAGS += -X "main.BuildTimestamp=$(shell date -u "+%Y-%m-%d %H:%M:%S")"
LDFLAGS += -X "main.airVersion=$(AIRVER)"
LDFLAGS += -X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

GO := GO111MODULE=on CGO_ENABLED=0 go
GOLANGCI_LINT_VERSION = v1.61.0

.PHONY: init
init: install-golangci-lint
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Install pre-commit hook"
	@ln -s $(shell pwd)/hooks/pre-commit $(shell pwd)/.git/hooks/pre-commit || true
	@chmod +x ./hack/check.sh

.PHONY: install-golangci-lint
install-golangci-lint:
ifeq (, $(shell which golangci-lintx))
	@$(shell curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION))
endif

.PHONY: setup
setup: init
	git init

.PHONY: check
check:
	@./hack/check.sh ${scope}

.PHONY: ci
ci: init
	@$(GO) mod tidy && $(GO) mod vendor

.PHONY: build
build: check
	$(GO) build -ldflags '$(LDFLAGS)'

.PHONY: install
install: check
	@echo "Installing air..."
	@$(GO) install -ldflags '$(LDFLAGS)'

.PHONY: release
release: check
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o bin/darwin/air
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o bin/linux/air
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o bin/windows/air.exe

.PHONY: docker-image
docker-image:
	docker build -t cosmtrek/air:$(AIRVER) -f ./Dockerfile .

.PHONY: push-docker-image
push-docker-image:
	docker push cosmtrek/air:$(AIRVER)
