# syntax=docker/dockerfile:1
FROM golang:1.21-alpine
RUN go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
RUN cp `which ginkgo` /ginkgo

## Base image consists of

#  FROM golang:1.21-alpine
#  RUN apk update;
#  RUN apk --no-cache add gcc musl-dev gcompat
#  RUN apk --no-cache add ca-certificates git curl

FROM kubeshop/testkube-ginkgo-executor:base
COPY ginkgo /bin/runner

WORKDIR /root/
COPY --from=0 /ginkgo /bin/ginkgo
RUN go version && ginkgo version
ENTRYPOINT ["/bin/runner"]
