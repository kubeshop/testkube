# syntax=docker/dockerfile:1
FROM golang:1.18-alpine
WORKDIR contrib/executor/ginkgo/build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd contrib/executor/ginkgo/cmd/agent;go build -o /runner -mod mod -a .
RUN go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo
RUN cp `which ginkgo` ginkgo

FROM golang:1.18-alpine
RUN apk update;
RUN apk --no-cache add gcc musl-dev gcompat
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
COPY --from=0 /runner /bin/runner
COPY --from=0 /build/ginkgo /bin/ginkgo
RUN go version
RUN ginkgo version
ENTRYPOINT ["/bin/runner"]
