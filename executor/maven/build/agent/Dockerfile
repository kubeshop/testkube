# syntax=docker/dockerfile:1
FROM golang:1.17 as builder

WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd cmd/agent;go build -o /runner -mod mod -a .

FROM maven:3.8.4-jdk-11
COPY --from=builder /runner /bin/runner
# do no overwrite the entrypoint
CMD ["/bin/runner"]
