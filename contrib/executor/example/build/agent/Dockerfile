# syntax=docker/dockerfile:1
FROM golang:1.18
WORKDIR contrib/executor/example/build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd contrib/executor/example/cmd/agent;go build -o /runner -mod mod -a .

FROM alpine
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
COPY --from=0 /runner /bin/runner
ENTRYPOINT ["/bin/runner"]
