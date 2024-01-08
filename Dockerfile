FROM golang:1.21.5-alpine3.19 as builder

RUN apk --update --no-cache add git build-base

ADD . /go/src/github.com/OSSystems/cdn
WORKDIR /go/src/github.com/OSSystems/cdn/cmd

RUN go mod edit -dropreplace github.com/OSSystems/cdn && \
    go mod edit -require=github.com/OSSystems/cdn@master && \
    go get github.com/OSSystems/cdn@master && \
    go mod tidy && \
    go build

FROM alpine:3.19

RUN apk --update --no-cache add ca-certificates

COPY --from=builder /go/src/github.com/OSSystems/cdn/cmd/cmd /usr/local/bin/cdn

ENTRYPOINT ["/usr/local/bin/cdn"]
