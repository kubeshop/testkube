# syntax=docker/dockerfile:1
FROM golang:1.18 as builder

# build k6 0.36.0 with prometheus support
ENV K6_VERSION=v0.36.0
RUN go install go.k6.io/xk6/cmd/xk6@v0.6.1 && xk6 build --with github.com/grafana/xk6-output-prometheus-remote@latest

WORKDIR /build

ENV CGO_ENABLED=0 
ENV GOOS=linux
COPY . .
RUN cd cmd/agent;go build -o /build/runner -mod mod -a .

FROM loadimpact/k6:0.36.0
WORKDIR /home/k6
COPY --from=builder /go/k6 /usr/bin/k6
COPY --from=builder /build/runner /bin/runner
ENTRYPOINT ["/bin/runner"]
