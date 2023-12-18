FROM golang:1.21 as builder
# # build k6 0.36.0 with prometheus support
ENV K6_VERSION=v0.48.0
RUN go install go.k6.io/xk6/cmd/xk6@v0.10.0 && xk6 build $K6_VERSION --with github.com/grafana/xk6-output-prometheus-remote@latest

# syntax=docker/dockerfile:1
FROM grafana/k6:0.48.0
COPY k6 /bin/runner
WORKDIR /home/k6
COPY --from=builder /go/k6 /usr/bin/k6
ENTRYPOINT ["/bin/runner"]
