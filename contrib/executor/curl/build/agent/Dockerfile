FROM kubeshop/testkube-executor-wrapper:0.0.1
COPY curl /bin/runner
# TODO(emil): for testing this should be enabled through the job templates
ENV ISTIO_PROXY_WAIT="true"
ENTRYPOINT ["/bin/wrapper.sh"]
