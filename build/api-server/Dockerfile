# syntax=docker/dockerfile:1
FROM golang:1.17
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0 
ENV GOOS=linux

ARG VERSION 
ARG COMMIT 

RUN cd cmd/api-server;go build -ldflags "-X github.com/kubeshop/testkube/internal/pkg/api.Version=$VERSION -X github.com/kubeshop/testkube/internal/pkg/api.Commit=$COMMIT" -X github.com/kubeshop/testkube/pkg/telemetry.telemetryToken=${{ secrets.TELEMETRY_TOKEN }} -o /app -mod mod -a .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates libssl1.1
# TODO remove when alpine will get it - We need upgrade libssl to fix 
RUN apk upgrade libssl1.1
WORKDIR /root/
COPY --from=0 /app /bin/app
EXPOSE 8088
CMD ["/bin/app"]
