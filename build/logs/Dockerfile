# syntax=docker/dockerfile:1
ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE}
RUN apk --no-cache add ca-certificates libssl1.1 git skopeo
WORKDIR /root/
COPY testkube-logs-server /bin/app
USER 1001
EXPOSE 8088
ENTRYPOINT ["/bin/app"]
