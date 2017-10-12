LDFLAGS += -X "main.BuildTimestamp=$(shell date -u "+%Y-%m-%d %H:%M:%S")"
LDFLAGS += -X "main.Version=$(shell git rev-parse HEAD)"

.PHONY: setup
setup:
	git init
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/golang/lint/golint
	go get -u github.com/Masterminds/glide
	glide init
	@echo "Install pre-commit hook"
	ln -s $(shell pwd)/hooks/pre-commit $(shell pwd)/.git/hooks/pre-commit

.PHONY: check
check:
	@./hack/check.sh ${scope}

.PHONY: ci
ci: setup check

.PHONY: build
	go build -ldflags '$(LDFLAGS)'
