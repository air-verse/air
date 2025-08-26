FROM golang:1.25 AS builder

LABEL maintainer="Rick Yu <cosmtrek@gmail.com>"

ENV GOPATH /go
ENV GO111MODULE on

COPY . /go/src/github.com/air-verse/air
WORKDIR /go/src/github.com/air-verse/air

RUN --mount=type=cache,target=/go/pkg/mod go mod download

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build make ci && make install

FROM golang:1.25

COPY --from=builder /go/bin/air  /go/bin/air

ENTRYPOINT ["/go/bin/air"]
