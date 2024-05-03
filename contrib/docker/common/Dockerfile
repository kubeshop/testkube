FROM alpine:latest as builder

# TODO(emil): build this through CI

RUN apk --no-cache add bash curl
COPY wrapper.sh /bin/wrapper.sh

ENTRYPOINT ["/bin/wrapper.sh"]
