FROM golang

MAINTAINER Rick Yu <cosmtrek@gmail.com>

ENV GOPATH /go

ADD . /go/src/github.com/cosmtrek/air
WORKDIR /go/src/github.com/cosmtrek/air
RUN make ci && make install

ENTRYPOINT ["/go/bin/air"]