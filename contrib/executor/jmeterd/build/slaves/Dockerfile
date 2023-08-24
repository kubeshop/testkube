FROM justb4/jmeter:5.5


EXPOSE 1099 60001
ENV SSL_DISABLED true

ADD plugins/ ${JMETER_CUSTOM_PLUGINS_FOLDER}
ADD lib/ ${JMETER_HOME}/lib
COPY ./scripts/jmeter-slaves.sh /jmeter_slaves_entrypoint.sh
RUN chmod +x /jmeter_slaves_entrypoint.sh

ENTRYPOINT /jmeter_slaves_entrypoint.sh