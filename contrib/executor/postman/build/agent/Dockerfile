FROM postman/newman
COPY postman /bin/runner
RUN apk --no-cache add ca-certificates git
WORKDIR /root/
EXPOSE 8082

USER 1001 

ENTRYPOINT ["/bin/runner"]
