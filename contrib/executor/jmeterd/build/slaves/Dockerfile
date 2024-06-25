FROM  kubeshop/jmeter:5.5

# support logs permissions
RUN mkdir -p /logs
RUN chmod 777 /logs

EXPOSE 1099 60001
ENV SSL_DISABLED true

COPY ./contrib/executor/jmeterd/scripts/jmeter-slaves.sh /jmeter_slaves_entrypoint.sh
COPY ./contrib/executor/jmeterd/scripts/jmeter-server.sh /jmeter-server.sh
RUN chmod +x /jmeter_slaves_entrypoint.sh
ENTRYPOINT /jmeter_slaves_entrypoint.sh