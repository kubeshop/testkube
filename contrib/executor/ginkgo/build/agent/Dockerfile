# syntax=docker/dockerfile:1
FROM golang:1.21-alpine as base

RUN apk update;
RUN go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
RUN cp `which ginkgo` /bin/ginkgo
RUN apk --no-cache add gcc musl-dev gcompat ca-certificates git curl

COPY ginkgo /bin/runner
WORKDIR /root/

RUN go version && ginkgo version

ENTRYPOINT ["/bin/runner"]
