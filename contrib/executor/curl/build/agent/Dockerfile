# syntax=docker/dockerfile:1
FROM golang:1.18
WORKDIR contrib/executor/curl/build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd contrib/executor/curl/cmd/agent;go build -o /runner -mod mod -a .

# TODO add valid dockerfile with executor
FROM alpine
RUN apk --no-cache add ca-certificates && \
    apk --no-cache add curl git
WORKDIR /root/
COPY --from=0 /runner /bin/runner

USER 1001

ENTRYPOINT ["/bin/runner"]
