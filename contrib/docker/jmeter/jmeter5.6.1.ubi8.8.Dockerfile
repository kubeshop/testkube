# Use Red Hat's Universal Base Image 8
FROM redhat/ubi8-minimal:8.8

# Labels and authorship
LABEL org.opencontainers.image.title="JMeter"                                          \
      org.opencontainers.image.description="Red Hat UBI with Java 11 and JMeter 5.6.1" \
      org.opencontainers.image.version="5.6.1"                                         \
      org.opencontainers.image.maintainer="support@testkube.io"                        \
      org.opencontainers.image.vendor="testkube"                                       \
      org.opencontainers.image.url="https://cloud.testkube.io"                         \
      org.opencontainers.image.source="https://github.com/kubeshop/testkube/tree/develop/contrib/docker/jmeter"

# Update the system and install required libraries
RUN microdnf update -y                              && \
    microdnf install curl unzip java-11-openjdk tar && \
    microdnf clean all

# Install JMeter 5.6.1
ENV JMETER_VERSION=5.6.1
RUN curl -L https://archive.apache.org/dist/jmeter/binaries/apache-jmeter-$JMETER_VERSION.tgz | tar xz -C /opt/ && \
    mv /opt/apache-jmeter-$JMETER_VERSION /opt/jmeter

# Set JMeter Home and add JMeter bin directory to the PATH
ENV JMETER_HOME /opt/jmeter
ENV PATH $JMETER_HOME/bin:$PATH

# Expose the required JMeter ports
EXPOSE 60000

# Command to run JMeter tests
ENTRYPOINT [ "jmeter" ]
