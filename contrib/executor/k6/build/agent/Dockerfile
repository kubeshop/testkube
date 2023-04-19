#Base image with adding pmotheus support: 0.2.4
# installed packages: go.k6.io/xk6/cmd/xk6@v0.6.1 && xk6 build --with github.com/grafana/xk6-output-prometheus-remote@latest
FROM kubeshop/testkube-k6-executor:base as builder
ENV K6_VERSION=v0.36.0

FROM loadimpact/k6:0.36.0
COPY k6 /bin/runner
WORKDIR /home/k6
COPY --from=builder /go/k6 /usr/bin/k6
ENTRYPOINT ["/bin/runner"]
