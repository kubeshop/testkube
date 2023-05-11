# syntax=docker/dockerfile:1
FROM golang:1.20.4
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd cmd/agent;go build -o /runner -mod mod -a .

FROM owasp/zap2docker-stable:2.12.0
ENV ZAP_HOME /zap
COPY --from=0 /runner /bin/runner
ENTRYPOINT ["/bin/runner"]