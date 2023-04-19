FROM golang:1.18-alpine
RUN go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
RUN cp `which ginkgo` /ginkgo

FROM golang:1.18-alpine
COPY ginkgo /bin/runner

RUN apk update;
RUN apk --no-cache add gcc musl-dev gcompat
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
COPY --from=0 /ginkgo /bin/ginkgo
RUN go version && ginkgo version
ENTRYPOINT ["/bin/runner"]
