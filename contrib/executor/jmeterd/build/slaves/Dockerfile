FROM  kubeshop/jmeter:5.5


EXPOSE 1099 60001
ENV SSL_DISABLED true

COPY ./contrib/executor/jmeterd/scripts/jmeter-slaves.sh /jmeter_slaves_entrypoint.sh
RUN chmod +x /jmeter_slaves_entrypoint.sh
ENTRYPOINT /jmeter_slaves_entrypoint.sh