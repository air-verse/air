FROM golang:1.10

MAINTAINER Rick Yu <cosmtrek@gmail.com>

ENV GOPATH /go

COPY . /go/src/github.com/cosmtrek/air
WORKDIR /go/src/github.com/cosmtrek/air
RUN make ci && make install

ENTRYPOINT ["/go/bin/air"]