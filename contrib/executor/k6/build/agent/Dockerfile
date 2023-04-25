# Base image with adding prometheus support version 0.2.4

# FROM golang:1.18 as builder
# RUN go install go.k6.io/xk6/cmd/xk6@v0.9.0 && xk6 build $K6_VERSION --with github.com/grafana/xk6-output-prometheus-remote@latest

FROM kubeshop/testkube-k6-executor:base as builder
ENV K6_VERSION=v0.36.0

FROM loadimpact/k6:0.36.0
COPY k6 /bin/runner
WORKDIR /home/k6
COPY --from=builder /go/k6 /usr/bin/k6
ENTRYPOINT ["/bin/runner"]