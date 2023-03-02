# syntax=docker/dockerfile:1
FROM golang:1.19
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

RUN cd cmd/agent;go build -o /runner -mod mod -a .

FROM justb4/jmeter
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
COPY --from=0 /runner /bin/runner
ENTRYPOINT ["/bin/runner"]
