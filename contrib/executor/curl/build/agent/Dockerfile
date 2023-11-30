# syntax=docker/dockerfile:1

# This version of curl built with the following configure options
# --enable-static --disable-ldap --enable-ipv6 --enable-unix-sockets --with-ssl --with-libssh2 --with-nghttp2=/usr \
# --prefix=/usr/local --with-gssapi

FROM curlimages/curl
COPY curl /bin/runner
WORKDIR /root/
USER 1001
ENTRYPOINT ["/bin/runner"]
