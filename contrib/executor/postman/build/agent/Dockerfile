# syntax=docker/dockerfile:1
FROM postman/newman
COPY postman /bin/runner
RUN apk --no-cache add ca-certificates git

# support npm reports
RUN mkdir -p /.npm
RUN chmod 777 /.npm
RUN chmod 777 /root/
RUN chmod 777 /usr/local/lib/node_modules
RUN chmod 777 /usr/local/bin

WORKDIR /root/
EXPOSE 8082

USER 1001 

ENTRYPOINT ["/bin/runner"]
