FROM golang:1.17

MAINTAINER Rick Yu <cosmtrek@gmail.com>

ENV GOPATH /go
ENV GO111MODULE on

COPY . /go/src/github.com/cosmtrek/air
WORKDIR /go/src/github.com/cosmtrek/air
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN make ci && make install

ENTRYPOINT ["/go/bin/air"]
