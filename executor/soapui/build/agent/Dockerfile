# syntax=docker/dockerfile:1
FROM golang:1.18 as builder
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN cd cmd/agent;go build -o /runner -mod mod -a .

FROM smartbear/soapuios-testrunner
COPY --from=builder /runner /bin/runner
ENTRYPOINT ["/bin/runner"]
