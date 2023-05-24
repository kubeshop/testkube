# syntax=docker/dockerfile:1
FROM $ALPINE_IMAGE
RUN apk --no-cache add ca-certificates libssl1.1 git
WORKDIR /root/
COPY testkube-api-server /bin/app
USER 1001
EXPOSE 8088
ENTRYPOINT ["/bin/app"]
