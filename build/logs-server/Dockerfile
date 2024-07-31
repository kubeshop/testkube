ARG ALPINE_IMAGE
FROM ${ALPINE_IMAGE} as certs
# hadolint ignore=DL3018
RUN apk --no-cache --update add ca-certificates && (rm -rf /var/cache/apk/* || 0)

WORKDIR /root/

RUN mkdir /data
RUN chown -R 1001:0 /data

COPY testkube-logs-server /bin/app
USER 1001
EXPOSE 8088
ENTRYPOINT ["/bin/app"]
