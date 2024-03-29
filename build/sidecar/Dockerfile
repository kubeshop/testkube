# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates 
WORKDIR /root/
COPY testkube-logs-sidecar /bin/app
USER 1001
EXPOSE 8088
ENTRYPOINT ["/bin/app"]
