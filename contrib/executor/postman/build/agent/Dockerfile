# syntax=docker/dockerfile:1
FROM postman/newman:6.0.0-alpine

COPY postman /bin/runner
RUN npm install -g newman
RUN apk --no-cache add ca-certificates git curl

# support npm report permissions
RUN mkdir -p /.npm
RUN chmod 777 /.npm
RUN chmod 777 /root/
RUN chmod 777 /usr/local/lib/node_modules
RUN chmod 777 /usr/local/bin

WORKDIR /root/
EXPOSE 8082

USER 1001 

ENTRYPOINT ["/bin/runner"]
