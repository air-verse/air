AIRVER := $(shell git describe --tags)
LDFLAGS += -X "main.BuildTimestamp=$(shell date -u "+%Y-%m-%d %H:%M:%S")"
LDFLAGS += -X "main.airVersion=$(AIRVER)"
LDFLAGS += -X "main.goVersion=$(shell go version | sed -r 's/go version go(.*)\ .*/\1/')"

GO := GO111MODULE=on go

.PHONY: init
init:
	go install golang.org/x/lint/golint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Install pre-commit hook"
	@ln -s $(shell pwd)/hooks/pre-commit $(shell pwd)/.git/hooks/pre-commit || true
	@chmod +x ./hack/check.sh

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
