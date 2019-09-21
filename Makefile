LDFLAGS += -X "main.BuildTimestamp=$(shell date -u "+%Y-%m-%d %H:%M:%S")"
LDFLAGS += -X "main.Version=$(shell git rev-parse HEAD)"

GO := GO111MODULE=on go

.PHONY: init
init:
	go get -u golang.org/x/lint/golint
	go get -u golang.org/x/tools/cmd/goimports
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
	$(GO) build -ldflags '$(LDFLAGS)' ./cmd/air

.PHONY: install
install: check
	@echo "Installing..."
	@$(GO) install -ldflags '$(LDFLAGS)' ./cmd/air

.PHONY: release
release: check
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o bin/darwin/air ./cmd/air/
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o bin/linux/air ./cmd/air
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags '$(LDFLAGS)' -o bin/windows/air.exe ./cmd/air

.PHONY: docker-image
docker-image:
	docker build -t cosmtrek/air:v1.10 -f ./Dockerfile .
