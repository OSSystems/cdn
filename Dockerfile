FROM golang:1.21.5-alpine3.19 as builder

ADD . /go/src/github.com/OSSystems/cdn
WORKDIR /go/src/github.com/OSSystems/cdn/cmd

RUN go build

FROM alpine:3.19

RUN apk --update --no-cache add ca-certificates

COPY --from=builder /go/src/github.com/OSSystems/cdn/cmd/cmd /usr/local/bin/cdn

ENTRYPOINT ["/usr/local/bin/cdn"]
