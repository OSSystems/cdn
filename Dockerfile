FROM golang:1.21.5-alpine3.19 as builder

RUN apk add build-base
RUN apk --update --no-cache add git openssh make
RUN go get -u github.com/Masterminds/glide

ADD . /go/src/github.com/OSSystems/cdn
WORKDIR /go/src/github.com/OSSystems/cdn

RUN glide install
RUN go build

FROM alpine:3.19

RUN apk --update --no-cache add ca-certificates

COPY --from=builder /go/src/github.com/OSSystems/cdn/cdn /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/cdn"]
